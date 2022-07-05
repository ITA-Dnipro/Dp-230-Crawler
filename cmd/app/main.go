package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"parabellum.crawler/internal/crawler"
	"parabellum.crawler/internal/pubsub"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(pathToEnvFile)
	if err != nil {
		log.Panicln("Error loading .env file: ", err)
	}
}

func main() {
	app := new(Config)

	app.initPubSub()
	defer app.closePubSub()

	exitCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run := true
	for run {
		taskInfo, err := app.Consumer.ReadMessage(exitCtx)
		if err != nil {
			if err == exitCtx.Err() {
				log.Println("Exiting on termination signal")
				return
			}
			log.Panicf("Error consuming message: %v. %v\n", taskInfo, err)
		}

		siteID := app.idFromMessage(taskInfo)
		payload, err := taskInfo.Value.ConsumePayload()
		if err != nil {
			continue
		}
		siteName := payload.URL
		testsToComplete := payload.ForwardTo

		providedURL, err := url.Parse(siteName)
		if err != nil {
			log.Printf("Wrong site name provided %s: %v\n", siteName, err)
			continue
		}

		ctx, cancel := context.WithTimeout(exitCtx, DEFAULT_TIMEOUT)

		app.Crawler = crawler.NewCrawler(ctx, providedURL)
		app.Crawler.MaxJumps = MAX_DEPTH
		app.Crawler.SetNumberOfThreads(NUM_OF_THREADS)
		log.Printf("Visiting: %s, with %d max jumps & %d threads.\n",
			providedURL.String(), MAX_DEPTH, NUM_OF_THREADS)
		app.Crawler.ExploreLink(crawler.NewLink(providedURL.String()))
		app.Crawler.Wait()

		cancel()

		app.publishCompletedResults(exitCtx, siteID, testsToComplete)

		log.Printf("Done with task %s\tlink:\t%s.\n", siteID, siteName)

		select {
		case <-exitCtx.Done():
			log.Println("Exiting on termination signal")
			run = false
		default:
			continue
		}
	}

}

func (app *Config) initPubSub() error {
	kafkaURL := os.Getenv("KAFKA_URL")

	app.Consumer = pubsub.NewConsumer(kafkaURL, os.Getenv("KAFKA_TOPIC_API"))

	app.Producers = map[string]*pubsub.Producer{}
	for testName := range TestsFilters {
		app.Producers[testName] = pubsub.NewProducer(kafkaURL, testName)
	}

	return nil
}

func (app *Config) closePubSub() {
	app.Consumer.Close()
	for _, prod := range app.Producers {
		prod.Close()
	}
}

func (app *Config) idFromMessage(message pubsub.Message) string {
	return message.Value.ID
}

func (app *Config) distributeResultsBetweenTests(tests []string) map[string][]string {
	resForTests := map[string][]string{}
	app.Crawler.Result.Range(func(link, value any) bool {
		if curResponse, ok := value.(*crawler.Response); ok {
			for _, tName := range tests {
				tParam := TestsFilters[tName]
				if curResponse.EqualsByParams(tParam) {
					resForTests[tName] = append(resForTests[tName], curResponse.VisitedLink.URL)
				}
			}
		}
		return true
	})
	return resForTests
}

func (app *Config) publishCompletedResults(ctx context.Context, mainTaskID string, tests []string) {
	resForTests := app.distributeResultsBetweenTests(tests)

	for tName, tTask := range resForTests {
		app.Producers[tName].PublicMessage(ctx, pubsub.NewMessageProduce(mainTaskID, tTask))
	}
}

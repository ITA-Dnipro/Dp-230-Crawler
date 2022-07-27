package main

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"parabellum.crawler/internal/crawler"
	"parabellum.crawler/internal/model"
	"parabellum.crawler/internal/pubsub"
)

//Config represents the core application structure
type Config struct {
	Crawler   *crawler.Crawler            //crawler to visit links on a given url
	Consumer  *pubsub.Consumer            //to read tasks for the app from pubsub
	Producers map[string]*pubsub.Producer //to push tasks for test-services
}

func main() {
	app := new(Config)

	app.initPubSub()
	defer app.closePubSub()

	exitCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for {
		err := app.ExecuteNextTask(exitCtx)
		if err != nil {
			continue
		}

		select {
		case <-exitCtx.Done():
			log.Println("Exiting on termination signal")

			return
		default:
		}
	}
}

//ExecuteNextTask completes one received task from pubsub, returns error if something goes wrong
func (app *Config) ExecuteNextTask(exitCtx context.Context) error {
	taskInfo, err := app.Consumer.FetchMessage(exitCtx)
	if err != nil {
		if err == exitCtx.Err() {
			return nil
		}
		log.Printf("Error consuming message from:\t%s.\t%v\n", app.Consumer.Topic, err)

		return err
	}

	ctx, cancel := context.WithTimeout(exitCtx, EnvVarOfType("CRAWLER_DEFAULT_TIMEOUT", TypeTimeSecond).(time.Duration))
	err = app.doCrawlerJob(ctx, taskInfo.Value.URL)
	cancel()
	if err != nil {
		return err
	}

	err = app.publishCompletedResults(exitCtx, taskInfo.Value.ID, taskInfo.Value.ForwardTo)
	if err != nil {
		log.Printf("Not enough tests will be invoked for task ID: %s \t%v\n", taskInfo.Value.ID, err)

		return err
	}

	err = app.Consumer.CommitMessage(exitCtx, taskInfo)
	if err != nil {
		log.Printf("Failed to commit task ID: %s \t%v\n", taskInfo.Value.ID, err)
	} else {
		log.Printf("Done with task ID: %s\n", taskInfo.Value.ID)
	}

	return err
}

func (app *Config) initPubSub() {
	kafkaURL := EnvVarOfType("KAFKA_URL", TypeString).(string)
	topicRead := EnvVarOfType("KAFKA_TOPIC_API", TypeString).(string)

	app.Consumer = pubsub.NewConsumer(pubsub.RealKafkaReader(kafkaURL, topicRead), topicRead)

	app.Producers = map[string]*pubsub.Producer{}
	for testName := range TestsFilters {
		app.Producers[testName] = pubsub.NewProducer(pubsub.RealKafkaWriter(kafkaURL, testName), testName)
	}
}

func (app *Config) closePubSub() {
	_ = app.Consumer.Close()
	for _, prod := range app.Producers {
		_ = prod.Close()
	}
}

func (app *Config) doCrawlerJob(ctx context.Context, siteName string) error {
	providedURL, err := url.Parse(siteName)
	if err != nil {
		log.Printf("Wrong site name provided %s: %v\n", siteName, err)

		return err
	}
	app.Crawler = crawler.NewCrawler(ctx, providedURL)
	app.Crawler.MaxJumps = EnvVarOfType("CRAWLER_MAX_DEPTH", TypeInt).(int)
	app.Crawler.SetNumberOfThreads(EnvVarOfType("CRAWLER_NUM_OF_THREADS", TypeInt).(int))
	log.Printf("Crawling on: %s.\n", providedURL.String())
	app.Crawler.ExploreLink(crawler.NewLink(providedURL.String()))
	app.Crawler.Wait()

	return nil
}

func (app *Config) distributeResultsBetweenTests(tests []string) map[string][]string {
	resForTests := map[string][]string{}
	app.Crawler.Result.Range(func(link, value any) bool {
		if curResponse, ok := value.(*crawler.Response); ok {
			for _, tName := range tests {
				tParam := TestsFilters[tName]
				if curResponse.HasEqualParamsWith(tParam) {
					resForTests[tName] = append(resForTests[tName], curResponse.VisitedLink.URL)
				}
			}
		}

		return true
	})

	return resForTests
}

func (app *Config) publishCompletedResults(ctx context.Context, mainTaskID string, tests []string) error {
	resForTests := app.distributeResultsBetweenTests(tests)

	var errCount int
	for tName, tTask := range resForTests {
		err := app.Producers[tName].PublicMessage(ctx, model.NewMessageProduce(mainTaskID, tTask))
		if err != nil {
			log.Printf("Error publishing task for\t%s:\t%v\n", tName, err)
			errCount++
		}
	}

	if errCount > len(resForTests)/2 {
		return errors.New("failed to publish completed results")
	}

	return nil
}

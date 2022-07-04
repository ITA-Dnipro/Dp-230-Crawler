package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"parabellum.crawler/internal/crawler"
	"parabellum.crawler/internal/pubsub"

	"github.com/joho/godotenv"
)

const pathToEnvFile = ".env"

const (
	DEFAULT_TIMEOUT = time.Minute
	NUM_OF_THREADS  = 50
	MAX_DEPTH       = 3
)

type Config struct {
	Crawler   *crawler.Crawler
	Consumer  *pubsub.Consumer
	Producers []*pubsub.Producer
}

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
			log.Panicf("Error consuming message: %v. %v\n", taskInfo, err)
		}

		siteName := app.getUrlFromMessage(taskInfo)
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

		app.pubCompleted(exitCtx)

		select {
		case <-exitCtx.Done():
			run = false
		default:
			continue
		}
	}

}

func (app *Config) initPubSub() error {
	kafkaURL := os.Getenv("KAFKA_URL")

	app.Consumer = pubsub.NewConsumer(kafkaURL, os.Getenv("KAFKA_TOPIC_API"))

	producerSQLi := pubsub.NewProducer(kafkaURL, os.Getenv("KAFKA_TOPIC_TEST_SQLI"))
	app.Producers = append(app.Producers, producerSQLi)

	return nil
}

func (app *Config) closePubSub() {
	app.Consumer.Close()
	for _, prod := range app.Producers {
		prod.Close()
	}
}

func (app *Config) getUrlFromMessage(message pubsub.Message) string {
	return message.Value
}

func (app *Config) pubCompleted(ctx context.Context) {
	app.Crawler.Result.Range(func(link, value any) bool {
		if curResult, ok := value.(*crawler.Response); ok {
			strResult := fmt.Sprintf("Code %d:\tWith form: %t \t-\t%s\n",
				curResult.StatusCode, curResult.HasFormTag, link)

			for _, prod := range app.Producers {
				prod.PublicMessage(ctx, pubsub.NewMessage(strResult))
			}
		}
		return true
	})
}

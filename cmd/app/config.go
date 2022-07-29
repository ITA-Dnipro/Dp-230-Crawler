package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"parabellum.crawler/internal/crawler"
	"parabellum.crawler/internal/model"
	"parabellum.crawler/internal/network"
	"parabellum.crawler/internal/pubsub"
)

const (
	TypeString = iota
	TypeInt
	TypeTimeSecond
)

type TestTopicName string

const (
	Topic_5XX  = TestTopicName("5XX-check")
	Topic_XSS  = TestTopicName("XSS-check")
	Topic_SQLI = TestTopicName("SQLI-check")
	Topic_BA   = TestTopicName("BA-check")
	Topic_LFI  = TestTopicName("LFI-check")
)

var envDefaults = map[string]string{
	"KAFKA_URL":               "localhost:9092",
	"KAFKA_TOPIC_API":         "API-Service-Message",
	"CRAWLER_DEFAULT_TIMEOUT": "60",
	"CRAWLER_NUM_OF_THREADS":  "50",
	"CRAWLER_MAX_DEPTH":       "5",
	"GRPC_ADDR":               ":9090",
}

//Config represents the core application structure
type Config struct {
	Crawler    *crawler.Crawler                   //crawler to visit links on a given url
	Consumer   *pubsub.Consumer                   //to read tasks for the app from pubsub
	Producers  map[TestTopicName]*pubsub.Producer //to push tasks for test-services
	ClientGrpc *network.ClientGRPC                //to push 5xx errors directly to result collector
}

func init() {
	err := setEnvDefaults()
	if err != nil {
		log.Panicln("Error setting default env parameters")
	}
}

func setEnvDefaults() error {
	var err error
	for env, val := range envDefaults {
		if _, ok := os.LookupEnv(env); !ok {
			err = os.Setenv(env, val)
		}
		if err != nil {
			break
		}
	}

	return err
}

//TestFilters represents how endpoints should be filtered between tests
var TestsFilters = map[TestTopicName][crawler.NumOfBodyParams]bool{
	Topic_SQLI: fillTestFilter("SQLi" /*HasFormTag*/, true /*HasQueryParameter*/, false /*HasStatusError*/, false),
	Topic_BA:   fillTestFilter("Broken auth", true, false, false),
	Topic_XSS:  fillTestFilter("XSS", true, true, false),
	Topic_LFI:  fillTestFilter("LFI", false, true, false),
	Topic_5XX:  fillTestFilter("5xx", false, false, true),
}

func fillTestFilter(testName string, filters ...bool) [crawler.NumOfBodyParams]bool {
	if len(filters) != crawler.NumOfBodyParams {
		log.Panicln("Wrong filters number for test", testName)
	}

	var result [crawler.NumOfBodyParams]bool
	for i := 0; i < crawler.NumOfBodyParams; i++ {
		result[i] = filters[i]
	}

	return result
}

//EnvVarOfType returns environment variable converted to a given type
func EnvVarOfType(varName string, varType int) any {
	strVal := os.Getenv(varName)
	switch varType {
	case TypeString:
		return strVal
	case TypeInt:
		res, _ := strconv.Atoi(strVal)

		return res
	case TypeTimeSecond:
		res, _ := strconv.Atoi(strVal)

		return time.Duration(res) * time.Second
	}

	return nil
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
	err = app.doCrawlerJob(ctx, taskInfo.Value.URL, taskInfo.Value.SkipCrawler)
	cancel()
	if err != nil {
		return err
	}

	err = app.publishCompletedResults(exitCtx, taskInfo.Value.ID, taskInfo.Value.ForwardTo)
	if err != nil {
		log.Printf("Problems dealing with task ID: %s \t%v\n", taskInfo.Value.ID, err)
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

	app.Producers = map[TestTopicName]*pubsub.Producer{}
	for testName := range TestsFilters {
		topicName := string(testName)
		kafkaWriter := pubsub.RealKafkaWriter(kafkaURL, topicName)
		app.Producers[testName] = pubsub.NewProducer(kafkaWriter, topicName)
	}
}

func (app *Config) closePubSub() {
	_ = app.Consumer.Close()
	for _, prod := range app.Producers {
		_ = prod.Close()
	}
}

func (app *Config) doCrawlerJob(ctx context.Context, siteName string, skipCrawling bool) error {
	providedURL, err := url.Parse(siteName)
	if err != nil {
		log.Printf("Wrong site name provided %s: %v\n", siteName, err)

		return err
	}
	app.Crawler = crawler.NewCrawler(ctx, providedURL)
	app.Crawler.MaxJumps = EnvVarOfType("CRAWLER_MAX_DEPTH", TypeInt).(int)
	if skipCrawling {
		app.Crawler.MaxJumps = 0
	}
	app.Crawler.SetNumberOfThreads(EnvVarOfType("CRAWLER_NUM_OF_THREADS", TypeInt).(int))
	log.Printf("Crawling on: %s.\n", providedURL.String())
	app.Crawler.ExploreLink(crawler.NewLink(providedURL.String()))
	app.Crawler.Wait()

	if skipCrawling {
		app.Crawler.Result.Range(func(key, value any) bool {
			if curResponse, ok := value.(*crawler.Response); ok {
				for i := 0; i < crawler.NumOfBodyParams; i++ {
					curResponse.BodyParams[i] = true
				}
			}

			return true
		})
	}

	return nil
}

func (app *Config) distributeResultsBetweenTests(tests []string) (map[TestTopicName][]string, []*crawler.Response) {
	var responses5xx []*crawler.Response
	resForTests := map[TestTopicName][]string{}

	app.Crawler.Result.Range(func(link, value any) bool {
		if curResponse, ok := value.(*crawler.Response); ok {
			for _, tName := range tests {
				topic := TestTopicName(tName)
				tParam := TestsFilters[topic]

				if curResponse.HasEqualParamsWith(tParam) {
					if topic == Topic_5XX {
						responses5xx = append(responses5xx, curResponse)

						continue
					}

					resForTests[topic] = append(resForTests[topic], curResponse.VisitedLink.URL)
				}
			}
		}

		return true
	})

	for _, tName := range tests {
		topic := TestTopicName(tName)
		if _, ok := resForTests[topic]; !ok {
			resForTests[topic] = []string{}
		}
	}

	return resForTests, responses5xx
}

func (app *Config) publishCompletedResults(ctx context.Context, mainTaskID string, tests []string) error {
	var err error
	resForTests, responses5xx := app.distributeResultsBetweenTests(tests)

	for tName, tTask := range resForTests {
		if tName == Topic_5XX {
			err = app.ClientGrpc.Push5XXResult(ctx, mainTaskID, responses5xx)
		} else {
			err = app.Producers[tName].PublicMessage(ctx, model.NewMessageProduce(mainTaskID, tTask))
		}
		if err != nil {
			log.Printf("Error publishing task for\t%s:\t%v\n", tName, err)
		}
	}

	return err
}

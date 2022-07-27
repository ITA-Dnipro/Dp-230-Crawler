package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"parabellum.crawler/internal/crawler"
)

const (
	TypeString = iota
	TypeInt
	TypeTimeSecond
)

var envDefaults = map[string]string{
	"KAFKA_URL":               "kafka:9092",
	"KAFKA_TOPIC_API":         "API-Service-Message",
	"CRAWLER_DEFAULT_TIMEOUT": "60",
	"CRAWLER_NUM_OF_THREADS":  "50",
	"CRAWLER_MAX_DEPTH":       "5",
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
var TestsFilters = map[string][crawler.NumOfBodyParams]bool{
	"SQLI-check": fillTestFilter("SQLi" /*HasFormTag*/, true /*HasQueryParameter*/, false /*HasStatusError*/, false),
	"BA-check":   fillTestFilter("Broken auth", true, false, false),
	"XSS-check":  fillTestFilter("XSS", true, true, false),
	"LFI-check":  fillTestFilter("LFI", false, true, false),
	"5xx-check":  fillTestFilter("5xx", false, false, true),
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

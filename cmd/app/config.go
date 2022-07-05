package main

import (
	"log"
	"time"

	"parabellum.crawler/internal/crawler"
	"parabellum.crawler/internal/pubsub"
)

type Config struct {
	Crawler   *crawler.Crawler
	Consumer  *pubsub.Consumer
	Producers map[string]*pubsub.Producer
}

const pathToEnvFile = ".env"

const (
	DEFAULT_TIMEOUT = time.Minute
	NUM_OF_THREADS  = 50
	MAX_DEPTH       = 3
)

var TestsFilters = map[string][crawler.NumOfBodyParams]bool{
	"SQLI-check": fillTestFilter("SQLi", true, false),
	"BA-check":   fillTestFilter("Broken auth", true, false),
	"XSS-check":  fillTestFilter("XSS", true, true),
	"LFI-check":  fillTestFilter("LFI", false, true),
}

func fillTestFilter(testName string, filters ...bool) [crawler.NumOfBodyParams]bool {
	if len(filters) != crawler.NumOfBodyParams {
		log.Panicln("Wrong filters number for test", testName)
	}

	var result [crawler.NumOfBodyParams]bool
	result[crawler.HasFormTag] = filters[crawler.HasFormTag]
	result[crawler.HasQueryParameter] = filters[crawler.HasQueryParameter]
	return result
}

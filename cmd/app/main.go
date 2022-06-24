package main

import (
	"context"
	"log"
	"net/url"
	"time"
)

const mockSiteName = "http://httpstat.us/"
const DEFAULT_TIMEOUT = time.Minute

type Config struct {
	Crawler *Crawler
}

func main() {
	//receive input - site to crawl
	//TODO: unmock site name receiving, when get input data format
	providedURL, err := url.Parse(mockSiteName)
	if err != nil {
		log.Panicln("Wrong site name provided: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), DEFAULT_TIMEOUT)
	defer cancel()

	app := new(Config)
	app.Crawler = NewCrawlerInit(ctx, providedURL)
	//app.Crawler.SetNumberOfThreads(runtime.NumCPU() * 5)

	//extract endpoints from the site
	app.Crawler.ExploreLink(mockSiteName)
	app.Crawler.Wait()

	app.Crawler.Result.Range(func(_, val any) bool {
		res := val.(*Response)
		log.Printf("Status %d:\t%s\n", res.StatusCode, res.VisitedLink)

		return true
	})

	//filter results by tests specific
	//send result to kafka
}

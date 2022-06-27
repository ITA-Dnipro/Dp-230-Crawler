package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"
)

//TODO: remove const block after final implementation
const (
	mockSiteName = "https://kp.ru" //"https://fishki.net/"
	mockFileName = "results.log"
)

const (
	DEFAULT_TIMEOUT = time.Minute
	NUM_OF_THREADS  = 50
	MAX_DEPTH       = 3
)

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
	app.Crawler.MaxJumps = MAX_DEPTH
	app.Crawler.SetNumberOfThreads(NUM_OF_THREADS)

	//extract endpoints from the site
	log.Printf("Visiting: %s, with %d max jumps & %d threads.\n",
		providedURL.String(), MAX_DEPTH, NUM_OF_THREADS)
	app.Crawler.ExploreLink(NewLink(mockSiteName))
	app.Crawler.Wait()

	//filter results by tests specific

	//send result to kafka
	//TODO: replace with sending result to where it needed
	app.writeResultToFile()
}

func (app *Config) writeResultToFile() {
	file, err := os.Create(mockFileName)
	if err != nil {
		log.Panicln("Cannot create file for result: ", err)
	}
	defer file.Close()

	app.Crawler.Result.Range(func(link, value any) bool {
		curResult := value.(*Response)
		strResult := fmt.Sprintf("Code %d:\t%s\thas form: %t\n",
			curResult.StatusCode, link, curResult.HasFormTag)
		_, err := file.WriteString(strResult)
		if err != nil {
			log.Panicf("Error writing to file\t%s:\t%v", mockFileName, err)
		}

		return true
	})

	log.Println("Result saved to file: ", mockFileName)
}

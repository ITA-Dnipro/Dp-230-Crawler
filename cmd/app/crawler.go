package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const ERR_CONTEXT_DONE = "exit on context done"

type Crawler struct {
	URL    *url.URL
	Result *sync.Map
	ctx    context.Context
	ch     chan struct{}
	wg     *sync.WaitGroup
}

type Response struct {
	VisitedLink    string
	StatusCode     int
	BodyForQueries *goquery.Document
}

func NewCrawlerInit(ctx context.Context, urlCrawl *url.URL) *Crawler {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}

	return &Crawler{
		URL:    urlCrawl,
		Result: new(sync.Map),
		ctx:    ctx,
		wg:     new(sync.WaitGroup),
		ch:     ch,
	}
}

func (cr *Crawler) SetNumberOfThreads(num int) {
	ch := make(chan struct{}, num)
	ch <- struct{}{}
	cr.ch = ch
}

func (cr *Crawler) Wait() {
	cr.wg.Wait()
	close(cr.ch)
}

func (cr *Crawler) ExploreLink(link string) {
	cr.wg.Add(1)

	defer func() {
		<-cr.ch
		cr.wg.Done()
	}()

	if cr.shouldExit() || !cr.canVisitLink(link) {
		return
	}

	pageResponse, err := cr.makeGetRequest(link)
	if err != nil {
		return
	}

	cr.Result.Store(link, pageResponse)

	go cr.queueLinksVisit(pageResponse)
	cr.wg.Add(1)
}

func (cr *Crawler) queueLinksVisit(pageResponse *Response) {
	defer cr.wg.Done()

	links := cr.parseLinksFromResponse(pageResponse)
	if links == nil {
		return
	}

	for _, l := range links {
		if !cr.canVisitLink(l) {
			continue
		}

		select {
		case cr.ch <- struct{}{}:
			log.Println("\tin: ", l)
			go cr.ExploreLink(l)
		case <-cr.ctx.Done():
			return
		}
	}
}

func (cr *Crawler) canVisitLink(link string) bool {
	_, wasVisited := cr.Result.Load(link)

	rgxForHost := fmt.Sprintf(".*%s.*", strings.ReplaceAll(cr.URL.Host, ".", "\\."))
	isOurHost, _ := regexp.MatchString(rgxForHost, link)

	return !wasVisited && isOurHost
}

func (cr *Crawler) makeGetRequest(link string) (*Response, error) {
	if cr.shouldExit() {
		return nil, errors.New(ERR_CONTEXT_DONE)
	}

	client := new(http.Client)
	client.Timeout = 5 * time.Second
	resp, err := client.Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := &Response{
		VisitedLink: link,
		StatusCode:  resp.StatusCode,
	}

	if resp.StatusCode == http.StatusOK {
		queryDoc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return nil, err
		}
		result.BodyForQueries = queryDoc
	}

	return result, nil
}

func (cr *Crawler) parseLinksFromResponse(response *Response) []string {
	queryDoc := response.BodyForQueries
	if queryDoc == nil {
		return nil
	}

	var result []string
	queryDoc.Find(`[href]`).
		EachWithBreak(func(i int, sel *goquery.Selection) bool {
			link := cr.absoluteURL(sel.Text())
			result = append(result, link)

			return !cr.shouldExit()
		})

	return result
}

func (cr *Crawler) absoluteURL(u string) string {
	if strings.HasPrefix(u, "#") {
		return ""
	}

	absURL, err := cr.URL.Parse(u)
	if err != nil {
		return ""
	}

	absURL.Fragment = ""
	if absURL.Scheme == "//" {
		absURL.Scheme = cr.URL.Scheme
	}

	return absURL.String()
}

func (cr *Crawler) shouldExit() bool {
	select {
	case <-cr.ctx.Done():
		return true
	default:
		return false
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const ERR_CONTEXT_DONE = "exit on context done"

type Crawler struct {
	URL      *url.URL
	Result   *sync.Map
	MaxJumps int
	ctx      context.Context
	ch       chan struct{}
	wg       *sync.WaitGroup
}

func NewCrawlerInit(ctx context.Context, urlCrawl *url.URL) *Crawler {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}

	return &Crawler{
		URL:      urlCrawl,
		Result:   new(sync.Map),
		MaxJumps: 0,
		ctx:      ctx,
		wg:       new(sync.WaitGroup),
		ch:       ch,
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

func (cr *Crawler) ExploreLink(link *Link) {
	cr.wg.Add(1)

	defer func() {
		<-cr.ch
		cr.wg.Done()
	}()

	if cr.shouldExit() ||
		!cr.canVisitLink(link.URL) ||
		cr.MaxJumps < link.Jumps {
		return
	}

	cr.Result.Store(link.URL, &Response{})
	pageResponse, err := cr.makeGetRequest(link)
	if err != nil {
		cr.Result.Delete(link.URL) //???

		return
	}
	pageResponse.ContainsFormTag()
	cr.Result.Store(link.URL, pageResponse)

	go cr.queueLinksVisit(pageResponse)
	cr.wg.Add(1)
}

func (cr *Crawler) queueLinksVisit(pageResponse *Response) {
	defer cr.wg.Done()

	links := pageResponse.ParseLinksFromResponse(cr)
	if links == nil {
		return
	}

	for _, l := range links {
		if !cr.canVisitLink(l.URL) {
			continue
		}

		select {
		case cr.ch <- struct{}{}:
			{
				go cr.ExploreLink(l)
			}
		case <-cr.ctx.Done():
			return
		}
	}
}

func (cr *Crawler) canVisitLink(link string) bool {
	_, wasVisited := cr.Result.Load(link)

	rgxForHost := fmt.Sprintf("%s.*", strings.ReplaceAll(cr.URL.String(), ".", "\\."))
	isOurHost, _ := regexp.MatchString(rgxForHost, link)

	return !wasVisited && isOurHost
}

func (cr *Crawler) makeGetRequest(link *Link) (*Response, error) {
	if cr.shouldExit() {
		return nil, errors.New(ERR_CONTEXT_DONE)
	}

	//log.Println("\tin: ", link.URL)

	client := new(http.Client)
	client.Timeout = 7 * time.Second
	resp, err := client.Get(link.URL)
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

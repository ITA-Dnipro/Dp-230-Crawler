package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

const fakeLink = "https://this.is.link/"

var fakeHtmlBodyData = "<html><body><form action='/' method='get'>stub</form></body></html>"

type httpClientStub struct {
}

func (hCl *httpClientStub) Get(link string) (*http.Response, error) {
	fakeHtmlBody := ioutil.NopCloser(strings.NewReader(fakeHtmlBodyData))

	if link == fakeLink {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       fakeHtmlBody,
		}, nil
	}

	return nil, fmt.Errorf("error in GET request: ")
}

func init() {
	log.Println("Crawler suite init...")
}

func TestSetNumberOfThreads(t *testing.T) {
	crawler := NewCrawlerInit(context.Background(), &url.URL{})
	expectedNum := 5
	crawler.SetNumberOfThreads(expectedNum)
	require.EqualValuesf(t, expectedNum, cap(crawler.ch), "should set %d threads", expectedNum)
}

func TestWait(t *testing.T) {
	crawler := NewCrawlerInit(context.Background(), &url.URL{})
	crawler.Wait()
	for cc := range crawler.ch {
		_ = cc
	}
	_, ok := <-crawler.ch
	require.False(t, ok, "should close crawlers' chan")
}

func TestAbsoluteUrl(t *testing.T) {
	urlFake, _ := url.Parse(fakeLink)
	crawler := NewCrawlerInit(context.Background(), urlFake)

	tabTests := []struct {
		name           string
		valuePassed    string
		expectedResult string
	}{
		{
			name:           "link with prefix #",
			valuePassed:    "#1",
			expectedResult: "",
		},
		{
			name:           "incorrect link",
			valuePassed:    "=https://",
			expectedResult: "",
		},
		{
			name:           "correct link",
			valuePassed:    "?param1=1&param2=abc",
			expectedResult: fakeLink + "?param1=1&param2=abc",
		},
	}

	for _, test := range tabTests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedResult, crawler.absoluteURL(test.valuePassed), "should be equal")
		})
	}
}

func TestExploreLink(t *testing.T) {
	link := NewLink(fakeLink, 2)

	someResponse := &Response{
		VisitedLink: link,
		StatusCode:  http.StatusOK,
		HasFormTag:  true,
	}
	smWithResponse := &sync.Map{}
	smWithResponse.Store(fakeLink, someResponse)
	smWithEmpty := &sync.Map{}
	smWithEmpty.Store(fakeLink, &Response{})

	ctxCanCancel, cancel := context.WithCancel(context.Background())
	crawlerWithContext := NewCrawlerInit(ctxCanCancel, &url.URL{})
	cancel()

	crawlerWithLink := NewCrawlerInit(context.Background(), &url.URL{})
	crawlerWithLink.Result.Store(fakeLink, &Response{})

	crawlerWithMaxJumps := NewCrawlerInit(context.Background(), &url.URL{})
	crawlerWithMaxJumps.MaxJumps = 1

	urlGoogle, _ := url.Parse("https://www.google.com/")
	crawlerWithURL := NewCrawlerInit(context.Background(), urlGoogle)
	crawlerWithURL.MaxJumps = 10

	urlWrongGet, _ := url.Parse(fakeLink + "wrong")
	crawlerWithWrongGet := NewCrawlerInit(context.Background(), urlWrongGet)
	crawlerWithWrongGet.client = &httpClientStub{}

	urlCorrectGet, _ := url.Parse(fakeLink)
	crawlerWithCorrectGet := NewCrawlerInit(context.Background(), urlCorrectGet)
	crawlerWithCorrectGet.client = &httpClientStub{}
	crawlerWithCorrectGet.MaxJumps = 10

	tabTests := []struct {
		name     string
		expected *sync.Map
		link     *Link
		crawler  *Crawler
	}{
		{
			name:     "return on context",
			expected: &sync.Map{},
			link:     link,
			crawler:  crawlerWithContext,
		},
		{
			name:     "return on link exists",
			expected: smWithEmpty,
			link:     link,
			crawler:  crawlerWithLink,
		},
		{
			name:     "return on wrong host",
			expected: &sync.Map{},
			link:     link,
			crawler:  crawlerWithURL,
		},
		{
			name:     "return on max jumps",
			expected: &sync.Map{},
			link:     link,
			crawler:  crawlerWithMaxJumps,
		},
		{
			name:     "return on wrong GET",
			expected: &sync.Map{},
			link:     NewLink(crawlerWithWrongGet.URL.String()),
			crawler:  crawlerWithWrongGet,
		},
		{
			name:     "successfully crawled",
			expected: smWithResponse,
			link:     link,
			crawler:  crawlerWithCorrectGet,
		},
	}

	for _, test := range tabTests {
		t.Run(test.name, func(t *testing.T) {
			test.crawler.ExploreLink(test.link)
			requireSyncMapsAreEqual(t, test.expected, test.crawler.Result, "should return expected result")
		})
	}
}

func requireSyncMapsAreEqual(t *testing.T, a, b *sync.Map, msg string) {
	mapA := map[any]*Response{}
	mapB := map[any]*Response{}

	a.Range(func(key, value any) bool {
		resp, ok := value.(*Response)
		if ok {
			mapA[key] = resp
		}

		return true
	})
	b.Range(func(key, value any) bool {
		resp, ok := value.(*Response)
		if ok {
			mapB[key] = &Response{
				VisitedLink: resp.VisitedLink,
				StatusCode:  resp.StatusCode,
				HasFormTag:  resp.HasFormTag,
			}
		}

		return true
	})

	require.Equal(t, mapA, mapB, msg)
}

func TestQueueLinksVisit(t *testing.T) {
	urlFake, _ := url.Parse(fakeLink)
	crawler := NewCrawlerInit(context.Background(), urlFake)
	crawler.client = &httpClientStub{}
	crawler.MaxJumps = 10
	crawler.Result.Store(fakeLink+"search", &Response{})

	bodyWithLink := ioutil.NopCloser(strings.NewReader("<a href='search'>find</a><a href='/'>self</a>"))
	defer bodyWithLink.Close()
	queryWithLink, _ := goquery.NewDocumentFromReader(bodyWithLink)

	response := &Response{
		VisitedLink:    NewLink(fakeLink),
		StatusCode:     http.StatusOK,
		BodyForQueries: queryWithLink,
		HasFormTag:     true,
	}

	crawler.wg.Add(1)
	<-crawler.ch
	crawler.queueLinksVisit(response)
	time.Sleep(time.Millisecond)
	crawler.Wait()

	response.VisitedLink.Jumps++
	expectedSM := &sync.Map{}
	expectedSM.Store(fakeLink, response)
	expectedSM.Store(fakeLink+"search", &Response{})

	requireSyncMapsAreEqual(t, expectedSM, crawler.Result, "should be equal")
}

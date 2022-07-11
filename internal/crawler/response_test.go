package crawler

import (
	"context"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

func TestFillResponseBody(t *testing.T) {
	response := new(Response)
	body := ioutil.NopCloser(strings.NewReader("<html></html>"))
	defer body.Close()
	expected, expErr := goquery.NewDocumentFromReader(body)
	gotErr := response.FillResponseBody(body)
	require.Equal(t, expected, response.BodyForQueries, "body should equal")
	require.Equal(t, expErr, gotErr, "error should equal")
}

func TestContainsFormTag(t *testing.T) {
	bodyWithForm := ioutil.NopCloser(strings.NewReader("<form></form>"))
	defer bodyWithForm.Close()
	queryWithForm, _ := goquery.NewDocumentFromReader(bodyWithForm)

	tabTest := []struct {
		name     string
		response *Response
		expected bool
	}{
		{
			name: "with form",
			response: &Response{
				BodyForQueries: queryWithForm,
			},
			expected: true,
		},
		{
			name:     "empty response",
			response: &Response{},
			expected: false,
		},
	}

	for _, test := range tabTest {
		t.Run(test.name, func(t *testing.T) {
			test.response.fillHasFormTag()
			require.Equal(t, test.expected, test.response.BodyParams[HasFormTag], "should equal")
		})
	}
}

func TestParseLinksFromResponse(t *testing.T) {
	bodyWithLink := ioutil.NopCloser(strings.NewReader("<a href='https://www.google.com/'>google</a>"))
	defer bodyWithLink.Close()
	queryWithLink, _ := goquery.NewDocumentFromReader(bodyWithLink)

	bodyWithMultipleLinks := ioutil.NopCloser(strings.NewReader("<a href='/'></a><a href='search'></a>"))
	defer bodyWithMultipleLinks.Close()
	queryWithMultipleLinks, _ := goquery.NewDocumentFromReader(bodyWithMultipleLinks)
	urlForCrawler, _ := url.Parse("https://www.google.com/")

	tabTest := []struct {
		name     string
		crawler  *Crawler
		response *Response
		expected []*Link
	}{
		{
			name:    "with link",
			crawler: new(Crawler),
			response: &Response{
				BodyForQueries: queryWithLink,
			},
			expected: []*Link{NewLink("https://www.google.com/", 0)},
		},
		{
			name:    "with multiple links",
			crawler: NewCrawler(context.Background(), urlForCrawler),
			response: &Response{
				BodyForQueries: queryWithMultipleLinks,
			},
			expected: []*Link{
				NewLink("https://www.google.com/", 0),
				NewLink("https://www.google.com/search", 0),
			},
		},
		{
			name:     "empty response",
			crawler:  new(Crawler),
			response: &Response{},
			expected: nil,
		},
	}

	for _, test := range tabTest {
		t.Run(test.name, func(t *testing.T) {
			gotLinks := test.response.ParseLinksFromResponse(test.crawler)
			require.Equal(t, test.expected, gotLinks, "should equal")
		})
	}
}

func TestClearResponseBody(t *testing.T) {
	bodyWithForm := ioutil.NopCloser(strings.NewReader("<form></form>"))
	defer bodyWithForm.Close()
	queryWithForm, _ := goquery.NewDocumentFromReader(bodyWithForm)
	response := &Response{
		BodyForQueries: queryWithForm,
	}
	response.ClearResponseBody()
	require.Nil(t, response.BodyForQueries, "should be nil")
}

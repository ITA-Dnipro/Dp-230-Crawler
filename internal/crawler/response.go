package crawler

import (
	"io"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

const NumOfBodyParams = 2
const (
	HasFormTag = iota
	HasQueryParameter
)

type Response struct {
	VisitedLink    *Link
	StatusCode     int
	BodyForQueries *goquery.Document
	BodyParams     [NumOfBodyParams]bool
}

type Link struct {
	URL   string
	Jumps int
}

func NewLink(uri string, jumps ...int) *Link {
	jumpsToSet := 0
	if len(jumps) > 0 {
		jumpsToSet = jumps[0]
	}

	return &Link{
		URL:   uri,
		Jumps: jumpsToSet,
	}
}

func NewResponse(link *Link, status int) *Response {
	return &Response{
		VisitedLink: link,
		StatusCode:  status,
	}
}

func (resp *Response) EqualsByParams(comparedParams [NumOfBodyParams]bool) bool {
	return comparedParams == resp.BodyParams
}

func (resp *Response) FillResponseBody(receivedBody io.ReadCloser) error {
	queryDoc, err := goquery.NewDocumentFromReader(receivedBody)
	if err != nil {
		return err
	}
	resp.BodyForQueries = queryDoc

	return nil
}

func (resp *Response) FillResponseParameters() {
	resp.fillHasFormTag()
	resp.fillHasQueryParams()
}

func (resp *Response) fillHasFormTag() {
	queryDoc := resp.BodyForQueries
	if queryDoc == nil {
		return
	}

	if len(queryDoc.Has("form").Nodes) > 0 {
		resp.BodyParams[HasFormTag] = true
	}
}

func (resp *Response) fillHasQueryParams() {
	linkUrl, err := url.Parse(resp.VisitedLink.URL)
	if err != nil {
		return
	}
	params := linkUrl.Query()
	resp.BodyParams[HasQueryParameter] = len(params) > 0
}

func (resp *Response) ParseLinksFromResponse(crawl *Crawler) []*Link {
	queryDoc := resp.BodyForQueries
	if queryDoc == nil {
		return nil
	}

	var linkDepth int
	if resp.VisitedLink != nil {
		linkDepth = resp.VisitedLink.Jumps + 1
	}

	var result []*Link
	queryDoc.Find(`[href]`).
		EachWithBreak(func(i int, sel *goquery.Selection) bool {
			linkURL := crawl.absoluteURL(sel.Text())
			if len(sel.Nodes) > 0 {
				for _, attr := range sel.Nodes[0].Attr {
					if attr.Key == "href" {
						linkURL = crawl.absoluteURL(attr.Val)
					}
				}
			}

			result = append(result, NewLink(linkURL, linkDepth))

			return !crawl.shouldExit()
		})

	return result
}

func (resp *Response) ClearResponseBody() {
	resp.BodyForQueries = nil
}

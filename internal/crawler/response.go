package crawler

import (
	"io"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

//NumOfBodyParams size of array of test-services filter parameters
const NumOfBodyParams = 3
const (
	HasFormTag        = iota //index of HasForm filter in array of test-services parameters
	HasQueryParameter        //index of HasQuery filter
	HasStatusError
)

//Response representation of a single crawler result
type Response struct {
	VisitedLink    *Link                 //link that was visited
	StatusCode     int                   //http status code
	BodyForQueries *goquery.Document     //body for further analysis with goquery lib
	BodyParams     [NumOfBodyParams]bool //values with filter matching 0-has form, 1-has query param ...
}

//Link url to visit with jumps made to get to that url
type Link struct {
	URL   string //visited url
	Jumps int    //depth where this very link was found on
}

//NewLink is a [crawler.Link] constructor
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

//NewResponse is a [crawler.Response] constructor
func NewResponse(link *Link, status int) *Response {
	return &Response{
		VisitedLink: link,
		StatusCode:  status,
	}
}

//HasEqualParamsWith returns true if resp.BodyParams has at least one similar parameter to the given parameters, false - otherwise
func (resp *Response) HasEqualParamsWith(comparedParams [NumOfBodyParams]bool) bool {
	for i := 0; i < NumOfBodyParams; i++ {
		if resp.BodyParams[i] && resp.BodyParams[i] == comparedParams[i] {
			return true
		}
	}

	return false
}

//FillResponseBody transforms given parameter to a resp.BodyForQueries for further goquery processing
func (resp *Response) FillResponseBody(receivedBody io.ReadCloser) error {
	queryDoc, err := goquery.NewDocumentFromReader(receivedBody)
	if err != nil {
		return err
	}
	resp.BodyForQueries = queryDoc

	return nil
}

//FillResponseParameters calculates resp.BodyParams
func (resp *Response) FillResponseParameters() {
	resp.fillHasFormTag()
	resp.fillHasQueryParams()
	resp.fillHasStatusError()
}

func (resp *Response) fillHasStatusError() {
	resp.BodyParams[HasStatusError] = resp.StatusCode >= 500
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

//ParseLinksFromResponse returns array of new url-links found in a given resp.BodyForQueries
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

//ClearResponseBody deletes resp.BodyForQueries value for memory saving purposes
//used after resp.BodyForQueries is no longer needed
func (resp *Response) ClearResponseBody() {
	resp.BodyForQueries = nil
}

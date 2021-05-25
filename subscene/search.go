package subscene

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var intRE = regexp.MustCompile(`\d+`)

type SearchResults []*SearchResult

func (s SearchResults) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type SearchResult struct {
	URI   *url.URL
	Title string
	Subs  int
}

func (s *SearchResult) String() string { return s.Title }

func (api *API) Search(query string, retries int) (SearchResults, error) {
	req, err := http.NewRequest(
		"POST",
		uri("subtitles", "searchbytitle").String(),
		queryBody(url.Values{"query": []string{query}}),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := api.doReq(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	retry, err := shouldRetry(res, retries)
	if err != nil {
		return nil, err
	}
	if retry {
		return api.Search(query, retries-1)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	results := make(SearchResults, 0, 40)
	doc.Find(".search-result li").Each(func(i int, s *goquery.Selection) {
		a := s.Find(".title a")
		title := strings.TrimSpace(a.Text())
		hr, ok := a.Attr("href")
		if !ok {
			return
		}

		uri, err := href(hr)
		if err != nil {
			return
		}

		count := intRE.FindString(s.Find(".count").Text())
		subs, _ := strconv.Atoi(count)
		results = append(results, &SearchResult{uri, title, subs})
	})

	return results, nil
}

package subscene

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type API struct {
	c   *http.Client //= http.DefaultClient
	req chan struct{}
}

func New(c *http.Client) *API {
	if c == nil {
		c = http.DefaultClient
	}

	req := make(chan struct{})
	go func() {
		for {
			req <- struct{}{}
			time.Sleep(time.Millisecond * 300)
		}
	}()

	return &API{c, req}
}

const ua = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"

var absURLRE = regexp.MustCompile(`^https?:`)
var base, _ = url.Parse("https://subscene.com")

func safeReq(req *http.Request) *http.Request {
	req.Header.Set("User-Agent", ua)
	return req
}

func (api *API) doReq(req *http.Request) (*http.Response, error) {
	<-api.req
	return api.c.Do(safeReq(req))
}

func uri(paths ...string) *url.URL {
	u := *base
	u.Path = path.Join(paths...)
	return &u
}

func href(href string) (*url.URL, error) {
	uri := &url.URL{}
	*uri = *base
	var err error
	if len(href) == 0 {
		return uri, nil
	} else if absURLRE.MatchString(href) || strings.HasPrefix(href, "//") {
		uri, err = url.Parse(href)
		if err != nil || uri.Host != base.Host {
			return uri, err
		}
		if uri.Scheme == "" {
			uri.Scheme = base.Scheme
		}
		href = ""
	} else if href[0] == '/' {
		uri.Path = href
		href = ""
	}

	uri.Path = path.Join(uri.Path, href)
	return uri, nil
}

func queryBody(q url.Values) io.Reader { return strings.NewReader(q.Encode()) }

func shouldRetry(res *http.Response, retries int) (bool, error) {
	if res.StatusCode == http.StatusConflict {
		if retries > 0 {
			return true, nil
		}
		return false, errors.New("too many requests")
	}

	if res.StatusCode != http.StatusOK {
		return false, errors.New(res.Status)
	}

	return false, nil
}

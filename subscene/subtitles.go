package subscene

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/frizinak/subscene/archive"
)

type Downloads []*Download

func (d Downloads) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d Downloads) FilterLanguage(l Language) Downloads {
	dls := make(Downloads, 0, len(d))
	for _, dl := range d {
		if dl.Lang == l {
			dls = append(dls, dl)
		}
	}

	return dls
}

func (d Downloads) FilterN(n ...int) Downloads {
	dls := make(Downloads, 0, len(n))
	for _, ix := range n {
		if ix < 0 || ix >= len(d) {
			panic("out of bounds")
		}
		dls = append(dls, d[ix])
	}

	return dls
}

type Download struct {
	Lang    Language
	URI     *url.URL
	Title   string
	Author  string
	Comment string
	HI      bool
}

func (s *Download) String() string { return s.Title }

type Language string

const (
	LangEnglish              Language = "english"
	LangDutch                Language = "dutch"
	LangArabic               Language = "arabic"
	LangBengali              Language = "bengali"
	LangBig_5_code           Language = "big_5_code"
	LangBrazillianPortuguese Language = "brazillian"
	LangBurmese              Language = "burmese"
	LangChinese              Language = "chinese"
	LangCroatian             Language = "croatian"
	LangDanish               Language = "danish"
	LangEstonian             Language = "estonian"
	LangFarsi_persian        Language = "farsi_persian"
	LangFinnish              Language = "finnish"
	LangFrench               Language = "french"
	LangGerman               Language = "german"
	LangGreek                Language = "greek"
	LangHebrew               Language = "hebrew"
	LangIndonesian           Language = "indonesian"
	LangItalian              Language = "italian"
	LangJapanese             Language = "japanese"
	LangKorean               Language = "korean"
	LangLatvian              Language = "latvian"
	LangLithuanian           Language = "lithuanian"
	LangMalay                Language = "malay"
	LangMalayalam            Language = "malayalam"
	LangNorwegian            Language = "norwegian"
	LangPolish               Language = "polish"
	LangPortuguese           Language = "portuguese"
	LangRussian              Language = "russian"
	LangSerbian              Language = "serbian"
	LangSinhala              Language = "sinhala"
	LangSlovenian            Language = "slovenian"
	LangSpanish              Language = "spanish"
	LangSwedish              Language = "swedish"
	LangThai                 Language = "thai"
	LangTurkish              Language = "turkish"
	LangVietnamese           Language = "vietnamese"
)

func (api *API) subtitlePage(u *url.URL, retries int) (Downloads, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

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
		return api.subtitlePage(u, retries-1)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	dls := make(Downloads, 0, 100)
	doc.Find(".a1 a").Each(func(i int, s *goquery.Selection) {
		hr, ok := s.Attr("href")
		if !ok {
			return
		}
		uri, err := href(hr)
		if err != nil {
			return
		}

		lang := Language(path.Base(path.Dir(uri.Path)))
		text := strings.TrimSpace(s.Find("span").Last().Text())
		pp := s.Parent().Parent()
		author := pp.Find(".a5").Text()
		comment := pp.Find(".a6").Text()
		hi := pp.Find(".a41").Length() != 0
		dls = append(dls, &Download{lang, uri, text, author, comment, hi})
	})

	return dls, err
}

func (api *API) Subtitles(r *SearchResult, retries int) (Downloads, error) {
	return api.subtitlePage(r.URI, retries)
}

func (api *API) SubtitlePage(path string, retries int) (Downloads, error) {
	return api.subtitlePage(uri("subtitles", path), retries)
}

func (api *API) DownloadURI(d *Download, retries int) (*url.URL, error) {
	req, err := http.NewRequest("GET", d.URI.String(), nil)
	if err != nil {
		return nil, err
	}

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
		return api.DownloadURI(d, retries-1)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	hr, ok := doc.Find(".download a").Attr("href")
	if !ok {
		return nil, errors.New("missing download link")
	}

	return href(hr)
}

type ZipInfo struct {
	URI       *url.URL
	Filename  string
	Extracted map[string]string
	Err       error
}

func (api *API) Download(u *url.URL, dir, name string, retries int) ZipInfo {
	var z ZipInfo
	z.URI = u

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		z.Err = err
		return z
	}

	res, err := api.doReq(req)
	if err != nil {
		z.Err = err
		return z
	}
	defer res.Body.Close()

	retry, err := shouldRetry(res, retries)
	if err != nil {
		z.Err = err
		return z
	}
	if retry {
		return api.Download(u, dir, name, retries-1)
	}

	_, params, _ := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
	if v, ok := params["filename"]; ok {
		z.Filename = v
	}

	size, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		z.Err = err
		return z
	}
	if size > 1024*1024*80 {
		z.Err = errors.New("body too large")
		return z
	}

	mtype, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	arch, err := archive.NewReader(mtype, io.LimitReader(res.Body, size), size)
	if err != nil {
		z.Err = err
		return z
	}

	dest := dir
	if name != "" {
		dest = filepath.Join(dest, name) + ".srt"
	}

	z.Extracted, z.Err = arch.Extract(dest, func(f string) bool {
		return filepath.Ext(f) == ".srt"
	})

	_, _ = io.Copy(io.Discard, res.Body)

	return z
}

func (api *API) Get(d Downloads, dir, name string, retries int, cb func(ZipInfo)) error {
	var gerr error
	var wg sync.WaitGroup
	for _, dl := range d {
		wg.Add(1)
		go func(dl *Download) {
			defer wg.Done()
			uri, err := api.DownloadURI(dl, retries)
			if err != nil {
				gerr = err
				return
			}

			z := api.Download(uri, dir, name, retries)
			if cb != nil {
				cb(z)
			}

			if z.Err != nil {
				gerr = z.Err
				return
			}
		}(dl)
	}

	wg.Wait()
	return gerr
}

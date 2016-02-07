package crawl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type Crawler struct {
	depth int
	queue chan *Link
}

type Sitemap struct {
	entries map[string]map[string]bool
	mux     *sync.Mutex
}

type Link struct {
	url       string
	parentUrl string
	isAsset   bool
}

// Constructs a new Crawler.
func NewCrawler(depth int) *Crawler {
	return &Crawler{
		depth: depth,
		queue: make(chan *Link),
	}
}

// Constructs a new Sitemap.
func NewSitemap() *Sitemap {
	return &Sitemap{
		entries: make(map[string]map[string]bool),
		mux:     &sync.Mutex{},
	}
}

// Crawl crawls a start URL for all links and assets and builds
// a sitemap with pages and assets per crawled link. Links are
// restricted to the same domain but assets are not since they
// are likely to be served by a CDN.
func (c *Crawler) Crawl(url string) (*Sitemap, error) {
	sitemap := NewSitemap()

	wg := &sync.WaitGroup{}
	done := make(chan bool)
	seen := make(map[string]bool)

	parentDomain, err := getDomain(url)
	if err != nil {
		return nil, err
	}

	wg.Add(1)
	go visit(url, c.queue, wg)

	// Waits for all goroutines to finish and signals the fact to
	// the `done` channel in order to terminate the select loop.
	go func() {
		wg.Wait()
		done <- true
	}()

	for {
		select {
		case link := <-c.queue:
			linkDomain, err := getDomain(link.url)
			if err != nil {
				continue
			}

			// We allow assets to come from a different domain.
			if !link.isAsset && linkDomain != parentDomain {
				continue
			}

			sitemap.AddLink(link)

			// Ensures we don't visit URLs twice.
			if seen[link.url] {
				continue
			}
			seen[link.url] = true

			wg.Add(1)
			go visit(link.url, c.queue, wg)
		case <-done:
			return sitemap, nil
		}
	}
}

// Adds a `Link` to the sitemap in a thread-safe manner.
func (s *Sitemap) AddLink(link *Link) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, ok := s.entries[link.parentUrl]
	if !ok {
		s.entries[link.parentUrl] = make(map[string]bool)
	}
	s.entries[link.parentUrl][link.url] = link.isAsset
}

// A convenience method to pretty-print a sitemap.
func (s *Sitemap) PrettyPrint() {
	for parentUrl, children := range s.entries {
		if parentUrl == "" {
			continue
		}
		fmt.Printf("=> %s\n", parentUrl)
		for url, isAsset := range children {
			isAssetStr := "PAGE"
			if isAsset {
				isAssetStr = "ASSET"
			}
			fmt.Printf("  -> [%s] %s\n", isAssetStr, url)
		}
	}
}

// Visits a URL and enqueues extracted links for further processing.
func visit(url string, queue chan *Link, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	links := extractLinks(url, resp.Body)

	for _, link := range links {
		queue <- link
	}
}

// Transforms a URL to an absolute URL given its parent. If the
// URL is already an absolute URL (which could be in a different
// domain) it is returned as is.
func absURL(href, parent string) (string, error) {
	url, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	parentUrl, err := url.Parse(parent)
	if err != nil {
		return "", err
	}
	resolved := parentUrl.ResolveReference(url)
	return resolved.String(), nil
}

// Extracts an attribute from an HTML token. Returns an empty
// string if the attribute is not found.
func extractAttr(t html.Token, attr string) string {
	for _, a := range t.Attr {
		if a.Key == attr {
			return a.Val
		}
	}
	return ""
}

// Extracts and returns a list of absolute URLs (links and assets)
// from an HTML document. Accepts a reader as it is returned from
// the HTTP client.
func extractLinks(url string, body io.Reader) []*Link {
	links := make([]*Link, 0)
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			return links
		case tt == html.StartTagToken:
			t := z.Token()
			if t.Data == "a" || t.Data == "script" {
				// Extract anchor and script tags but discard everything
				// without an href since they could be broken links or
				// inline scripts.
				href := extractAttr(t, "href")
				if href == "" {
					continue
				}
				href, err := absURL(href, url)
				if err != nil {
					continue
				}
				links = append(links, &Link{href, url, t.Data == "script"})
			} else if t.Data == "link" {
				// Extract link tags but limit the set to just stylesheets.
				rel := extractAttr(t, "rel")
				href := extractAttr(t, "href")
				if href == "" || rel != "stylesheet" {
					continue
				}
				href, err := absURL(href, url)
				if err != nil {
					continue
				}
				links = append(links, &Link{href, url, true})
			}
		}
	}
	return links
}

// Given a URL it returns its domain.
func getDomain(href string) (string, error) {
	url, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	tokens := strings.Split(url.Host, ":")
	return tokens[0], nil
}

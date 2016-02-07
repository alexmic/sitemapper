package crawl

import (
	"log"
	"fmt"
	"net/http"
	"sync"
)

type Crawler struct {
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
func NewCrawler() *Crawler {
	return &Crawler{queue: make(chan *Link)}
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

	parentDomain, err := GetDomain(url)
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
			linkDomain, err := GetDomain(link.url)
			if err != nil {
				continue
			}

			// We allow assets to come from a different domain.
			if !link.isAsset && linkDomain != parentDomain {
				continue
			}

			sitemap.AddEntry(link.url, link.parentUrl, link.isAsset)

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
func (s *Sitemap) AddEntry(url, parentUrl string, isAsset bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, ok := s.entries[parentUrl]
	if !ok {
		s.entries[parentUrl] = make(map[string]bool)
	}
	s.entries[parentUrl][url] = isAsset
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
		log.Print("Error in fetching url: %s", err)
		return
	}
	defer resp.Body.Close()

	links := ExtractLinks(url, resp.Body)

	for _, link := range links {
		queue <- link
	}
}

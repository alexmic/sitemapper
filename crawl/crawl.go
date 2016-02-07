package crawl

import (
    "fmt"
    "io"
    "sync"
    "strings"
    "golang.org/x/net/html"
    "net/http"
    "net/url"
)

type Sitemap struct {
    entries map[string]map[string]bool
    mux *sync.Mutex
}

func newSitemap() *Sitemap {
    return &Sitemap{
        entries: make(map[string]map[string]bool),
        mux: &sync.Mutex{},
    }
}

func (s *Sitemap) AddLink(link *Link) {
    s.mux.Lock()
    defer s.mux.Unlock()
    _, ok := s.entries[link.parentUrl]
    if (!ok) {
        s.entries[link.parentUrl] = make(map[string]bool)
    }
    s.entries[link.parentUrl][link.url] = link.isAsset
}

func (s *Sitemap) PrettyPrint() {
    for parentUrl, children := range(s.entries) {
        if parentUrl == "" {
            continue
        }
        fmt.Printf("=> %s\n", parentUrl)
        for url, isAsset := range(children) {
            isAssetStr := "PAGE"
            if isAsset {
                isAssetStr = "ASSET"
            }
            fmt.Printf("  -> [%s] %s\n", isAssetStr, url)
        }
    }
}

type Link struct {
    url string
    parentUrl string
    isAsset bool
}

type Crawler struct {
    depth int
    queue chan *Link
}

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

func extractAttr(t html.Token, attr string) string {
    for _, a := range t.Attr {
        if a.Key == attr {
            return a.Val
        }
    }
    return ""
}

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

func getDomain(href string) (string, error) {
    url, err := url.Parse(href)
    if err != nil {
        return "", err
    }
    tokens := strings.Split(url.Host, ":")
    return tokens[0], nil
}

func NewCrawler(depth int) *Crawler {
    return &Crawler{
        depth: depth,
        queue: make(chan *Link),
    }
}

func (c *Crawler) Crawl(url string) (*Sitemap, error) {
    sitemap := newSitemap()

    wg := &sync.WaitGroup{}
    done := make(chan bool)
    seen := make(map[string]bool)

    parentDomain, err := getDomain(url)
    if (err != nil) {
        return nil, err
    }

    wg.Add(1)
    go visit(url, c.queue, wg)

    go func() {
        wg.Wait()
        done <- true
    }()

    for {
        select {
        case link := <- c.queue:
            linkDomain, err := getDomain(link.url)
            if err != nil {
                continue
            }

            if !link.isAsset && linkDomain != parentDomain {
                continue
            }

            sitemap.AddLink(link)

            if seen[link.url] {
                continue
            }
            seen[link.url] = true

            wg.Add(1)
            go visit(link.url, c.queue, wg)
        case <- done:
            return sitemap, nil
        }
    }
}
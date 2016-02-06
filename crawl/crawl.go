package crawl

import (
    "fmt"
    "io"
    "time"
    "golang.org/x/net/html"
    "net/http"
    "net/url"
)

type Link struct {
    url string
    parentUrl string
    isAsset bool
}

type Crawler struct {
    depth int
    all chan *Link
    seen chan *Link
    unseen chan *Link
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

func visit(url string, queue chan *Link) {
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println(err)
        return
    }
    defer resp.Body.Close()

    links := extractLinks(url, resp.Body)

    for _, link := range links {
        go func(link *Link) {
            queue <- link
        }(link)
    }
}

func getDomain(href string) (string, error) {
    url, err := url.Parse(href)
    if err != nil {
        return "", err
    }
    return url.Host, nil
}

func distributor(all, seen, unseen chan *Link) {
    seenMap := make(map[string]bool)
    for link := range all {
        if seenMap[link.url] {
            seen <- link
        } else {
            seenMap[link.url] = true
            unseen <- link
        }
    }
}

func NewCrawler(depth int) *Crawler {
    return &Crawler{
        depth: depth,
        all: make(chan *Link),
        seen: make(chan *Link),
        unseen: make(chan *Link),
    }
}

func (c *Crawler) Crawl(url string) {
    startLink := &Link{url: url}
    seen := 0

    parentDomain, err := getDomain(url)
    if (err != nil) {
        return
    }

    go func() {
        c.all <- startLink
    }()

    go distributor(c.all, c.seen, c.unseen)

    for {
        select {
        case link := <- c.unseen:
            linkDomain, err := getDomain(link.url)
            if err == nil && linkDomain == parentDomain {
                fmt.Println("VISITING", link.url)
                go visit(link.url, c.all)
            }
        case <- c.seen:
            seen++
        case <- time.After(10000 * time.Millisecond):
            return
        }
    }
}
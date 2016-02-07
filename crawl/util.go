package crawl

import (
	"golang.org/x/net/html"
	"io"
	"net/url"
	"strings"
)

// Transforms a URL to an absolute URL given its parent. If the
// URL is already an absolute URL (which could be in a different
// domain) it is returned as is.
func AbsURL(href, parent string) (string, error) {
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

// Extracts and returns a list of absolute URLs (links and assets)
// from an HTML document. Accepts a reader as it is returned from
// the HTTP client.
func ExtractLinks(url string, body io.Reader) []*Link {
	links := make([]*Link, 0)
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			return links
		case tt == html.StartTagToken:
			t := z.Token()
			isAsset := false
			href := ""
			if t.Data == "a" {
				href = extractAttr(t, "href")
			} else if t.Data == "script" {
				href = extractAttr(t, "src")
				isAsset = true
			} else if t.Data == "link" {
				// Extract link tags but limit the set to just stylesheets.
				rel := extractAttr(t, "rel")
				if rel != "stylesheet" {
					continue
				}
				href = extractAttr(t, "href")
				isAsset = true
			}
			if href == "" {
				continue
			}
			href, err := AbsURL(href, url)
			if err != nil {
				continue
			}
			links = append(links, &Link{href, url, isAsset})
		}
	}
	return links
}

// Given a URL it returns its domain.
func GetDomain(href string) (string, error) {
	url, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	tokens := strings.Split(url.Host, ":")
	return tokens[0], nil
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

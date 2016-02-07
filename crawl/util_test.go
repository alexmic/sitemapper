package crawl

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAbsURL_RelativePath(t *testing.T) {
	url, err := AbsURL("/path", "http://example.com")
	if assert.NoError(t, err) {
		assert.Equal(t, url, "http://example.com/path")
	}
}

func TestAbsURL_AbsolutePath(t *testing.T) {
	url, err := AbsURL("http://example2.com/path", "http://example.com")
	if assert.NoError(t, err) {
		assert.Equal(t, url, "http://example2.com/path")
	}
}

func TestExtractLinks(t *testing.T) {
	html :=
		`<html>
      <head>
        <link rel="stylesheet" href="//example.com/styles.css">
      </head>
      <body>
        <div>foo</div>
        <a href="http://example.com">example</a>
        <script src="//example.com/script.js"></script>
      </body>
    <html>`
	r := bytes.NewReader([]byte(html))
	links := ExtractLinks("http://example.com", r)
	assert.Equal(t, len(links), 3)
}

func TestExtractLinks_SkipNoHref(t *testing.T) {
	html :=
		`<html>
      <head>
        <link rel="stylesheet">
      </head>
      <body>
        <div>foo</div>
        <a>example</a>
      </body>
    <html>`
	r := bytes.NewReader([]byte(html))
	links := ExtractLinks("http://example.com", r)
	assert.Empty(t, links)
}

func TestExtractLinks_SkipNoSrc(t *testing.T) {
	html :=
		`<html>
      <head>
      </head>
      <body>
        <div>foo</div>
        <script>var foo="bar";</script>
      </body>
    <html>`
	r := bytes.NewReader([]byte(html))
	links := ExtractLinks("http://example.com", r)
	assert.Empty(t, links)
}

func TestExtractLinks_SkipNonStylesheets(t *testing.T) {
	html :=
		`<html>
      <head>
        <link rel="alternate" type="application/atom+xml" href="/blog/news/atom">
      </head>
      <body>
        <div>foo</div>
      </body>
    <html>`
	r := bytes.NewReader([]byte(html))
	links := ExtractLinks("http://example.com", r)
	assert.Empty(t, links)
}

func TestGetDomain(t *testing.T) {
	domain, err := GetDomain("http://example.com")
	if assert.NoError(t, err) {
		assert.Equal(t, domain, "example.com")
	}
}

func TestGetDomain_SkipPort(t *testing.T) {
	domain, err := GetDomain("http://example.com:3000")
	if assert.NoError(t, err) {
		assert.Equal(t, domain, "example.com")
	}
}

package main

import (
	"github.com/alexmic/sitemapper/crawl"
	"github.com/codegangsta/cli"
	"os"
    "log"
    "strings"
    "net/url"
)

func main() {
	app := cli.NewApp()

	app.Name = "sitemapper"
	app.Usage = "a crawler for building sitemaps"

	app.Action = func(c *cli.Context) {
		rawurl := c.Args().Get(0)
        if rawurl == "" {
            log.Fatal("Please give a URL to crawl.")
            os.Exit(1)
        }
        url, err := url.Parse(rawurl)
        if err != nil {
            log.Fatal(err)
            os.Exit(1)
        }
        if url.Scheme == "" {
            url.Scheme = "http"
        }
        absUrl := url.String()
        if !strings.HasSuffix(absUrl, "/") {
            absUrl += "/"
        }
		crawler := crawl.NewCrawler(2)
		sitemap, _ := crawler.Crawl(absUrl)
		sitemap.PrettyPrint()
	}

	app.Run(os.Args)
}

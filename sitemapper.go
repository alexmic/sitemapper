package main

import (
	"github.com/alexmic/sitemapper/crawl"
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()

	app.Name = "sitemapper"
	app.Usage = "a crawler for building sitemaps"

	app.Action = func(c *cli.Context) {
		url := c.Args().Get(0)
		crawler := crawl.NewCrawler(2)
		sitemap, _ := crawler.Crawl(url)
		sitemap.PrettyPrint()
	}

	app.Run(os.Args)
}

package crawl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSitemap_AddEntry(t *testing.T) {
	sitemap := NewSitemap()
	sitemap.AddEntry("http://foo.bar/1", "http://foo.bar", false)
	sitemap.AddEntry("http://foo.bar/2", "http://foo.bar", false)
	sitemap.AddEntry("http://foo.bar/3", "http://foo.bar/1", true)

	assert.Contains(t, sitemap.entries, "http://foo.bar")
	assert.Contains(t, sitemap.entries, "http://foo.bar/1")
	assert.NotContains(t, sitemap.entries, "http://foo.bar/2")
	assert.NotContains(t, sitemap.entries, "http://foo.bar/3")

	assert.Contains(t, sitemap.entries["http://foo.bar"], "http://foo.bar/1")
	assert.Contains(t, sitemap.entries["http://foo.bar"], "http://foo.bar/2")
	assert.Contains(t, sitemap.entries["http://foo.bar/1"], "http://foo.bar/3")
}

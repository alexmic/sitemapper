# sitemapper
A crawler which builds a sitemap for a given domain.

## Running

```
go get -u github.com/alexmic/sitemapper
go build
./sitemapper http://alexmic.net
```

The binary will print a sitemap of all pages on the domain along with the pages they
link to and their assets.
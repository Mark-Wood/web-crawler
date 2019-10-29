package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Page defines a node in the site-map tree structure
type Page struct {
	URL         *url.URL
	Parent      *Page
	FirstChild  *Page
	NextSibling *Page
}

// MaxPages limits the number of pages to construct
const MaxPages = 25

var pageCount int = 0

// Crawl print a textual site-map of a domain with the startURL as the root
func Crawl(startURL string) {
	url, err := url.Parse(startURL)

	if err != nil {
		// TODO: log error
	}

	page := &Page{URL: url}
	populateChildPages(page)

	printPage(page, 0)
}

func printPage(page *Page, depth int) {
	if page == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat(" ", depth), page.URL.String())
	printPage(page.FirstChild, depth+1)
	printPage(page.NextSibling, depth)
}

func populateChildPages(page *Page) {
	if pageCount >= MaxPages {
		return
	}
	pageCount++

	resp, err := http.Get(page.URL.String())
	defer resp.Body.Close()

	if err != nil || !isValidHTML(resp) {
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		// ...
	}

	links := parseLinks(doc)

	var prevChildPage *Page
	for _, linkURL := range links {

		if (linkURL.Scheme != "http" && linkURL.Scheme != "https" && linkURL.Scheme != "") ||
			(linkURL.Host != page.URL.Host && linkURL.Host != "") {
			continue
		}
		absolutelinkURL := page.URL.ResolveReference(&linkURL)

		rootPage := page
		for rootPage.Parent != nil {
			rootPage = rootPage.Parent
		}
		if urlExists(absolutelinkURL, rootPage) {
			continue
		}

		childPage := &Page{URL: absolutelinkURL, Parent: page}
		if prevChildPage != nil {
			prevChildPage.NextSibling = childPage
		}
		if page.FirstChild == nil {
			page.FirstChild = childPage
		}
		prevChildPage = childPage
	}

	for childPage := page.FirstChild; childPage != nil; childPage = page.NextSibling {
		populateChildPages(childPage)
	}
}

func parseLinks(node *html.Node) []url.URL {

	links := make(map[url.URL]bool)

	var f func(n *html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i := 0; i < len(n.Attr); i++ {
				attr := n.Attr[i]
				if attr.Key != "href" {
					continue
				}
				url, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				url.Fragment = ""
				links[*url] = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(node)

	linksarray := make([]url.URL, len(links))
	i := 0
	for link := range links {
		linksarray[i] = link
		i++
	}
	return linksarray
}

func urlExists(url *url.URL, page *Page) bool {
	if page != nil {
		if *page.URL == *url {
			return true
		}
		return urlExists(url, page.FirstChild) || urlExists(url, page.NextSibling)
	}
	return false
}

func isValidHTML(resp *http.Response) bool {
	if resp.StatusCode != 200 {
		return false
	}
	isContentHTML := false
	for i := 0; i < len(resp.Header["Content-Type"]) && !isContentHTML; i++ {
		fmt.Println(resp)
		if strings.HasPrefix(resp.Header["Content-Type"][i], "text/html") {
			isContentHTML = true
		}
	}
	if !isContentHTML {
		return false
	}

	return true
}

//func
/*
 * TODO:
 * - logging
 * - run javascript?
 * - scan for links in non-html responses
 * - handle forms
 * - parallelisation (channels?)
 * - error handling
 * - agent string?
 */

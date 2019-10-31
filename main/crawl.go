package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// Page defines a node in the site-map tree structure
type Page struct {
	URL         *url.URL
	Parent      *Page
	FirstChild  *Page
	NextSibling *Page
}

// Crawl constructs a site-map of a domain with the startURL as the root
func Crawl(startURL string, maxDepth int) (*Page, error) {
	url, err := url.Parse(startURL)

	if err != nil {
		return nil, err
	}

	page := &Page{URL: url}
	populateChildPages(page, maxDepth, 1, nil)

	return page, nil
}

// Print prints a textual representation of a page instance
func Print(page *Page) {
	print(page, 0)
}

func populateChildPages(page *Page, maxDepth, depth int, waitgroup *sync.WaitGroup) {
	if waitgroup != nil {
		defer waitgroup.Done()
	}

	// Send HEAD request to avoid potentially large downloads of non-HTML resources
	resp, err := http.Head(page.URL.String())
	if err != nil || !isSuccessHTMLResponse(resp) {
		return
	}

	resp, err = http.Get(page.URL.String())
	if err != nil || !isSuccessHTMLResponse(resp) {
		// Only search for links in successful HTML responses
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil || doc == nil {
		// Not valid HTML response content
		return
	}

	links := parseLinks(doc)
	resp.Body.Close()

	var prevChildPage *Page
	for _, linkURL := range links {
		if (linkURL.Scheme != "http" && linkURL.Scheme != "https" && linkURL.Scheme != "") ||
			(linkURL.Host != page.URL.Host && linkURL.Host != "") {
			continue
		}
		absoluteLinkURL := page.URL.ResolveReference(&linkURL)

		rootPage := page
		for rootPage.Parent != nil {
			rootPage = rootPage.Parent
		}

		if urlExistsInTree(absoluteLinkURL, rootPage) {
			continue
		}

		if maxDepth != -1 && depth >= maxDepth {
			return
		}

		childPage := &Page{URL: absoluteLinkURL, Parent: page}
		if prevChildPage != nil {
			prevChildPage.NextSibling = childPage
		}
		if page.FirstChild == nil {
			page.FirstChild = childPage
		}
		prevChildPage = childPage
	}

	var childWaitGroup sync.WaitGroup
	for childPage := page.FirstChild; childPage != nil; childPage = childPage.NextSibling {
		childWaitGroup.Add(1)
		go populateChildPages(childPage, maxDepth, depth+1, &childWaitGroup)
	}
	childWaitGroup.Wait()
}

func isSuccessHTMLResponse(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return false
	}

	isContentHTML := false
	for _, contentType := range resp.Header["Content-Type"] {
		if strings.HasPrefix(contentType, "text/html") {
			isContentHTML = true
			break
		}
	}
	if !isContentHTML {
		return false
	}

	return true
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

				// Ignore "#fragment" part of URL
				url.Fragment = ""

				links[*url] = true
			}
		}

		// Traverse the tree depth-first
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			f(child)
		}
	}
	f(node)

	linksArray := make([]url.URL, len(links))
	i := 0
	for link := range links {
		linksArray[i] = link
		i++
	}
	return linksArray
}

func urlExistsInTree(url *url.URL, page *Page) bool {
	return page != nil && (*page.URL == *url ||
		urlExistsInTree(url, page.FirstChild) ||
		urlExistsInTree(url, page.NextSibling))
}

func print(page *Page, depth int) {
	if page == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat(" ", depth), page.URL.String())
	print(page.FirstChild, depth+1)
	print(page.NextSibling, depth)
}

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
func Crawl(URL string, maxDepth int) (*Page, error) {
	parsedURL, err := url.Parse(URL)

	if err != nil {
		return nil, err
	}

	page := &Page{URL: parsedURL}
	populateChildPages(page, maxDepth, 1, nil)

	return page, nil
}

// Print prints a textual representation of a page instance
func Print(page *Page) {
	print(page, 0)
}

func populateChildPages(page *Page, maxDepth, depth int, waitGroup *sync.WaitGroup) {
	if waitGroup != nil {
		defer waitGroup.Done()
	}

	// Send HEAD request to avoid potentially large downloads of non-HTML resources
	response, err := http.Head(page.URL.String())
	if err != nil || !isSuccessHTMLResponse(response) {
		return
	}

	response, err = http.Get(page.URL.String())
	if err != nil || !isSuccessHTMLResponse(response) {
		// Only search for links in successful HTML responses
		return
	}

	doc, err := html.Parse(response.Body)
	if err != nil || doc == nil {
		// Not valid HTML response content
		return
	}

	links := mineLinks(doc)
	response.Body.Close()

	var prevChildPage *Page
	for linkURL := range links {
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

func isSuccessHTMLResponse(response *http.Response) bool {
	if response == nil {
		return false
	}

	if !(response.StatusCode >= 200 && response.StatusCode < 400) {
		return false
	}

	isContentHTML := false
	for _, contentType := range response.Header["Content-Type"] {
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

func mineLinks(node *html.Node) map[url.URL]bool {
	links := make(map[url.URL]bool)

	var mineLinksInNode func(n *html.Node)
	mineLinksInNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i := 0; i < len(n.Attr); i++ {
				attr := n.Attr[i]
				if attr.Key != "href" {
					continue
				}

				parsedURL, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}

				// Ignore "#fragment" part of URL
				parsedURL.Fragment = ""

				links[*parsedURL] = true
			}
		}

		// Traverse the tree depth-first
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			mineLinksInNode(child)
		}
	}
	mineLinksInNode(node)

	return links
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

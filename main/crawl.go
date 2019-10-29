package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Page struct {
	URL         *url.URL
	Parent      *Page
	FirstChild  *Page
	NextSibling *Page
}

const MaxRequests = 25

var RequestCount int = 0

func Crawl(startUrl string) {
	url, err := url.Parse(startUrl)

	if err != nil {
		// TODO: log error
	}

	page := &Page{URL: url}
	PopulatePageChildren(page)

	PrintPage(page, 0)
}

func PrintPage(page *Page, depth int) {
	if page == nil {
		return
	}
	fmt.Printf("%s%s\n", strings.Repeat(" ", depth), page.URL.String())
	PrintPage(page.FirstChild, depth+1)
	PrintPage(page.NextSibling, depth)
}

func PopulatePageChildren(page *Page) {
	if RequestCount > MaxRequests {
		return
	}
	RequestCount++
	resp, err := http.Get(page.URL.String())
	defer resp.Body.Close()

	if err != nil || !IsValidHtml(resp) {
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		// ...
	}

	links := GetLinks(doc)

	var prevChildPage *Page
	for _, linkUrl := range links {

		if (linkUrl.Scheme != "http" && linkUrl.Scheme != "https" && linkUrl.Scheme != "") ||
			(linkUrl.Host != page.URL.Host && linkUrl.Host != "") {
			continue
		}
		absoluteLinkUrl := page.URL.ResolveReference(&linkUrl)

		rootPage := page
		for rootPage.Parent != nil {
			rootPage = rootPage.Parent
		}
		if UrlParentExists(absoluteLinkUrl, rootPage) {
			continue
		}

		childPage := &Page{URL: absoluteLinkUrl, Parent: page}
		if prevChildPage != nil {
			prevChildPage.NextSibling = childPage
		}
		if page.FirstChild == nil {
			page.FirstChild = childPage
		}
		prevChildPage = childPage
	}

	for childPage := page.FirstChild; childPage != nil; childPage = page.NextSibling {
		PopulatePageChildren(childPage)
	}
}

func GetLinks(node *html.Node) []url.URL {

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

func UrlParentExists(url *url.URL, page *Page) bool {
	if page != nil {
		if *page.URL == *url {
			return true
		}
		return UrlParentExists(url, page.FirstChild) || UrlParentExists(url, page.NextSibling)
	}
	return false
}

func IsValidHtml(resp *http.Response) bool {
	if resp.StatusCode != 200 {
		return false
	}
	isContentHtml := false
	for i := 0; i < len(resp.Header["Content-Type"]) && !isContentHtml; i++ {
		fmt.Println(resp)
		if strings.HasPrefix(resp.Header["Content-Type"][i], "text/html") {
			isContentHtml = true
		}
	}
	if !isContentHtml {
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

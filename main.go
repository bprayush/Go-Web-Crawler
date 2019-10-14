package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/llimllib/loglevel"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Link (url, text, depth)
type Link struct {
	url   string
	text  string
	depth int
}

// HTTPError (original)
type HTTPError struct {
	original string
}

// MaxDepth @default: 2
var MaxDepth = 2

// LinkReader @parms resp *http.Response, depth int @return []Link
func LinkReader(resp *http.Response, depth int) []Link {
	page := html.NewTokenizer(resp.Body)
	links := []Link{}

	var start *html.Token
	var text string

	for {
		_ = page.Next()
		token := page.Token()

		if token.Type == html.ErrorToken {
			break
		}

		if start != nil && token.Type == html.TextToken {
			text = fmt.Sprintf("%s%s", text, token.Data)
		}

		if token.DataAtom == atom.A {
			switch token.Type {
			case html.StartTagToken:
				if len(token.Attr) > 0 {
					start = &token
				}
			case html.EndTagToken:
				if start == nil {
					log.Warnf("Link end found without start: %s\n", text)
					continue
				}
				link := NewLink(*start, text, depth)
				if link.Valid() {
					links = append(links, link)
					log.Debugf("Link Found %v\n", link)
				}

				start = nil
				text = ""
			}
		}
	}

	log.Debug(links)
	return links
}

// NewLink @parms tag html.Token, text string, depth int @return Link
func NewLink(tag html.Token, text string, depth int) Link {
	link := Link{
		text:  strings.TrimSpace(text),
		depth: depth,
	}

	for i := range tag.Attr {
		if tag.Attr[i].Key == "href" {
			link.url = strings.TrimSpace(tag.Attr[i].Val)
		}
	}

	return link
}

// String @returns string
func (link Link) String() string {
	spacer := strings.Repeat("\t", link.depth)
	return fmt.Sprintf("%s%s (%d) - %s\n", spacer, link.text, link.depth, link.url)
}

// Valid @returns boolcle
func (link Link) Valid() bool {
	if link.depth >= MaxDepth {
		return false
	}
	if len(link.text) == 0 {
		return false
	}
	if len(link.url) == 0 || strings.Contains(strings.ToLower(link.url), "javascript") {
		return false
	}

	return true
}

func (httpError HTTPError) Error() string {
	return httpError.original
}

func recurDownloader(url string, depth int) {
	page, err := downloader(url)
	if err != nil {
		log.Error(err)
		return
	}
	links := LinkReader(page, depth)

	for _, link := range links {
		fmt.Println(links)
		if depth+1 < MaxDepth {
			recurDownloader(link.url, depth+1)
		}
	}
}

func downloader(url string) (resp *http.Response, err error) {
	log.Debugf("Downloading %s\n", url)
	resp, err = http.Get(url)
	if err != nil {
		log.Debugf("Error: %s\n", err)
		return
	}

	if resp.StatusCode > 299 {
		err = HTTPError{
			fmt.Sprintf("Error (%d): %s\n", resp.StatusCode, url),
		}
		log.Debug(err)
		return
	}
	return
}

func main() {
	log.SetPriorityString("info")
	log.SetPrefix("crawler")

	log.Debug(os.Args)

	if len(os.Args) < 2 {
		log.Fatalln("Missing Url args")
	}

	recurDownloader(os.Args[1], 0)
}

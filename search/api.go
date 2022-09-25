package search

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"os"
	"strings"
)

const apiURL = "https://pkg.go.dev/search?limit=50&m=package&q="

type Package struct {
	Name        string  `json:"name,omitempty"`
	Path        string  `json:"path"`
	ImportCount int     `json:"import_count"`
	Synopsis    string  `json:"synopsis,omitempty"`
	Fork        bool    `json:"fork,omitempty"`
	Stars       int     `json:"stars,omitempty"`
	Score       float64 `json:"score,omitempty"`
}

type Response struct {
	Results []Package `json:"results"`
}

func doSearch(pkg string) ([]Package, error) {
	url := fmt.Sprintf("%s%s", apiURL, pkg)
	var resp, err = http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(os.Stderr, resp.Body)
		return nil, fmt.Errorf("failed to search package, server return code=%s", resp.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	doc.Find("div.SearchSnippet").Each(func(i int, selection *goquery.Selection) {
		path := removeQuote(selection.Find(".SearchSnippet-header-path").Text())
		desc := selection.Find(".SearchSnippet-synopsiss").Text()
		name := removeQuote(selection.RemoveFiltered(".SearchSnippet-header-path").Find("h2 a").Text())
		pkgs = append(pkgs, Package{
			Name:     name,
			Path:     path,
			Synopsis: desc,
		})
	})
	return pkgs, nil
}

var _replacer = strings.NewReplacer("(", "", ")", "", " ", "")

func removeQuote(s string) string {
	return _replacer.Replace(s)
}

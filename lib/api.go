package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const apiURL = "https://api.godoc.org/search?q="

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
	var res Response
	err = json.NewDecoder(resp.Body).Decode(&res)
	return res.Results, err
}

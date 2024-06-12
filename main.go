package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type SearchResult struct {
	ResultRank  int
	ResultUrl   string
	ResultTitle string
	ResultDesc  string
}

var googleDomains = map[string]string{
	// shortened for brevity
	"com": "https://www.google.com/search?q=",
	"in":  "https://www.google.co.in/search?q=",
}

var userAgents = []string{
	// shortened for brevity
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
}

func randomUserAgents() string {
	rand.Seed(time.Now().Unix())
	randomNum := rand.Int() % len(userAgents)
	return userAgents[randomNum]
}

func buildGoogleUrls(searchTerm string, languageCode string, countryCode string, pages int, count int) ([]string, error) {
	fmt.Println("Building Google URLs")
	toScrape := []string{}
	searchTerm = strings.Trim(searchTerm, " ")
	searchTerm = strings.Replace(searchTerm, " ", "+", -1)
	if googleBase, found := googleDomains[countryCode]; found {
		for i := 0; i < pages; i++ {
			start := i * count
			scrapeURL := fmt.Sprintf("%s%s&num=%d&hl=%s&start=%d&filter=0", googleBase, searchTerm, count, languageCode, start)
			toScrape = append(toScrape, scrapeURL)
		}
	} else {
		err := fmt.Errorf("country %s is currently not supported", countryCode)
		return nil, err
	}
	return toScrape, nil
}

func getScrapeClient(proxyString interface{}) *http.Client {
	fmt.Println("Setting up HTTP client")
	switch v := proxyString.(type) {
	case string:
		proxyUrl, _ := url.Parse(v)
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	default:
		return &http.Client{}
	}
}

func scrapeClientRequest(searchURL string, proxyString interface{}) (*http.Response, error) {
	fmt.Printf("Sending request to %s\n", searchURL)
	baseClient := getScrapeClient(proxyString)
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", randomUserAgents())
	res, err := baseClient.Do(req)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return nil, err
	}
	if res.StatusCode != 200 {
		err := fmt.Errorf("scraper gets a non 200 status code: %d", res.StatusCode)
		return nil, err
	}
	return res, nil
}

func googleResultParser(response *http.Response, rank int) ([]SearchResult, error) {
	fmt.Println("Parsing response")
	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}
	results := []SearchResult{}
	sel := doc.Find("div.g")
	rank++
	for i := range sel.Nodes {
		item := sel.Eq(i)
		linkTag := item.Find("a")
		link, _ := linkTag.Attr("href")
		titleTag := item.Find("h3")
		descTag := item.Find("span.st")
		title := titleTag.Text()
		desc := descTag.Text()
		link = strings.Trim(link, " ")

		if link != "" && link != "#" && !strings.HasPrefix(link, "/") {
			result := SearchResult{
				rank,
				link,
				title,
				desc,
			}
			results = append(results, result)
			rank++
		}
	}
	return results, nil
}

func GoogleScrape(searchTerm string, languageCode string, proxyString interface{}, countryCode string, pages int, count int, backoff int) ([]SearchResult, error) {
	fmt.Println("Starting Google Scrape")
	results := []SearchResult{}
	resultCounter := 0
	googlePages, err := buildGoogleUrls(searchTerm, languageCode, countryCode, pages, count)
	if err != nil {
		return nil, err
	}
	for _, page := range googlePages {
		res, err := scrapeClientRequest(page, proxyString)
		if err != nil {
			fmt.Printf("Request error: %v\n", err)
			continue
		}
		data, err := googleResultParser(res, resultCounter)
		if err != nil {
			fmt.Printf("Parsing error: %v\n", err)
			continue
		}
		resultCounter += len(data)
		for _, result := range data {
			results = append(results, result)
		}
		time.Sleep(time.Duration(backoff) * time.Second)
	}
	return results, nil
}

func main() {
	result, err := GoogleScrape("akhil sharma", "en", nil, "in", 1, 20, 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for _, res := range result {
			fmt.Println(res)
		}
	}
}

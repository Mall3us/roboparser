package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/chromedp/chromedp"
	"github.com/dlclark/regexp2"
)

type Site struct {
	URL string
}

type Result struct {
	URL    string
	Status int
	Length int
}

func main() {

	// Parse url flag
	url := flag.String("u", "", "Specify URL of the target")
	flag.Parse()

	if err := os.Mkdir("output", os.ModePerm); err != nil {
		fmt.Println(err)
	}

	fmt.Println("[*] Gathering: " + *url + "/robots.txt")
	// Retrieve robots.txt content
	resp, err := http.Get(*url + "/robots.txt")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	sb := string(body)
	// Parse robots.txt
	r, _ := regexp2.Compile("(?<=[Disa|A]llow: ).*", 0)
	robotsEndpoints := regexp2FindAllString(r, sb)
	// Make requests to each of robots.txt endpoints
	jobs := make(chan Site, 3)
	results := make(chan Result, 3)

	for w := 1; w <= 3; w++ {
		go CrawlRobotsEndpoints(w, jobs, results)
	}
	go func() {
		for _, _url := range robotsEndpoints {
			jobs <- Site{URL: *url + _url}
		}
		close(jobs)
	}()

	for a := 1; a <= len(robotsEndpoints); a++ {
		result := <-results
		HtmlOutput(result.URL, result.Status, result.Length)
		fmt.Println(result.URL + " -- " + strconv.Itoa(result.Status) + " -- " + strconv.Itoa(result.Length))
	}

}

func regexp2FindAllString(re *regexp2.Regexp, s string) []string {
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

// Crawl robots.txt
func CrawlRobotsEndpoints(wId int, jobs <-chan Site, results chan<- Result) {

	for url := range jobs {
		resp, err := http.Get(url.URL)
		if err != nil {
			fmt.Printf("Error:\n %s\n", err)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error:\n %s\n", err)
			}
			sb := len(string(body))
			go TakeScreenshot(url.URL)
			results <- Result{
				URL:    url.URL,
				Status: resp.StatusCode,
				Length: sb,
			}
		}

	}
	//return url + robotsEndpoints + " -- " + resp.Status + " -- " + strconv.Itoa(sb)
}

func TakeScreenshot(url string) {

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.FullScreenshot(&buf, 95),
	); err != nil {
		fmt.Println(err)
	}
	r, _ := regexp2.Compile("/", 0)
	re, _ := r.Replace(url, "-", 2, -1)
	outputName := "./output/" + re + ".jpg"
	if err := ioutil.WriteFile(outputName, buf, 0644); err != nil {
		fmt.Println(err)
	}
}

func HtmlOutput(url string, Status int, Length int) {
	r, _ := regexp2.Compile("/", 0)
	regUrl, _ := r.Replace(url, "-", 2, -1)

	if _, err := os.Stat("index.html"); err != nil {
		buf := []byte("<!DOCTYPE html><html><head><div class='topnav'><h1 style='text-align:center'>ROBOPARSER Report</h1></div><link rel='stylesheet' href='mystyle.css'><title>Roboparser report</title></head><body><div class='textdiv'><p>URL: " + url + "<br>Status: " + strconv.Itoa(Status) + "<br>Content-Length: " + strconv.Itoa(Length) + "</p></div><br><img src=./output/" + regUrl + ".jpg class='center'><br>")
		if error := ioutil.WriteFile("index.html", buf, 0644); error != nil {
			fmt.Println(error)
		}
	} else {
		buf := "<div class='textdiv'><p>" + url + "<br>Status: " + strconv.Itoa(Status) + "</h3><br>Content-Length: " + strconv.Itoa(Length) + "</div><br><img src=./output/" + regUrl + ".jpg class='center'><br>"
		f, _ := os.OpenFile("index.html", os.O_APPEND|os.O_WRONLY, 0644)
		defer f.Close()
		if _, error := f.WriteString(buf); error != nil {
			fmt.Println(error)
		}
	}
}

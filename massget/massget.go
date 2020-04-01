package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var response int

func fetch(url string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

        if response != 0 && resp.StatusCode == response {
                fmt.Println(url)
        } else {
	        fmt.Println(resp.StatusCode, url, strings.Join(resp.Header["Content-Type"], " "), resp.Header["Server"])
        }
	resp.Body.Close()
}

func main() {
	var method string
	var concurrency int

	flag.StringVar(&method, "method", "GET", "HTTP Method to use (GET/POST)")
        flag.IntVar(&response, "r", 0, "Return only URL's matching HTTP response code (0 = all)")
	flag.IntVar(&concurrency, "c", 20, "Number of concurrent requests (default: 20)")
	flag.Parse()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	urls := make(chan string)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
                        defer wg.Done()
			for u := range urls {
				fetch(u)
			}
		}()
	}

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		current := input.Text()
		if !strings.HasPrefix(current, "http://") && !strings.HasPrefix(current, "https://") {
			current = "https://" + current
		}

		urls <- current
	}

	close(urls)
	wg.Wait()
}

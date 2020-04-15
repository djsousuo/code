package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
        "net"
        "time"
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
                return
	}

        if response != 0 && resp.StatusCode == response {
                fmt.Println(url)
        } else {
	        fmt.Println(resp.StatusCode, url, resp.Header["Content-Type"], resp.Header["Server"])
        }
	resp.Body.Close()
}

func main() {
	var method string
	var concurrency int
        var timeout int

	flag.StringVar(&method, "method", "GET", "HTTP Method to use (GET/POST)")
        flag.IntVar(&response, "r", 0, "Return only URL's matching HTTP response code (0 = all)")
	flag.IntVar(&concurrency, "c", 20, "Number of concurrent requests (default: 20)")
        flag.IntVar(&timeout, "t", 10, "Timeout (seconds)")
	flag.Parse()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
        http.DefaultTransport.(*http.Transport).DialContext = (&net.Dialer{Timeout: time.Duration(time.Duration(timeout) * time.Second), KeepAlive: time.Second,}).DialContext

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

package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	//"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Params string `json:"params"`
}

func checkHost(host string, params string, method string, client *http.Client) {
	originCheck := true
	req, err := http.NewRequest(method, host, strings.NewReader(params))
	if err != nil {
		log.Fatal(err)
	}

	if originCheck == true {
		req.Header.Set("Origin", "evil.com")
	}
	req.Header.Set("PHPSESSID", "t2l9le1hd9bvahp6qfjfqa7fk3")
	/*
		        req.Header.Set("PHPSESSID", "gm3lb9d2fi28tvna4sncteb043")
			req.Header.Set("Host", "movsx.dev")
	*/

	resp, err := client.Do(req)

	if resp != nil {
		if method == "POST" && resp.StatusCode == 200 {
			fmt.Println("[*] VULNERABLE (CSRF): " + host)
		}
		if resp.StatusCode == 405 {
			fmt.Println("[*] Bad request: " + host)
		}
		/*
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
				bodyStr := string(bodyBytes)
		*/

		origin := resp.Header.Get("Access-Control-Allow-Origin")
		if origin == "evil.com" || origin == "*" || origin == "*.evil.com" {
			if resp.Header.Get("Access-Control-Allow-Credentials") == "true" {
				fmt.Println("[*] Vulnerable (CORS): " + host)
			}
		}
	}
}

func main() {
	var err error
	hostInput := os.Stdin

	if len(os.Args) > 1 {
		hostInput, err = os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	}

	timeout := time.Duration(1000000 * 1000000)
	var transport = &http.Transport{
		MaxIdleConns:      30,
		IdleConnTimeout:   time.Second,
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	queue := make(chan string)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			for entry := range queue {
				/* GET https://host.com param1=a&param2=b */
				method, URL, params := "", "", ""
				tmp := strings.Split(entry, " ")
				if len(tmp) < 2 {
					continue
				}

				if len(tmp) == 2 {
					method, URL, params = tmp[0], tmp[1], ""
				} else {
					method, URL, params = tmp[0], tmp[1], tmp[2]
				}
				checkHost(URL, params, method, client)

			}
			wg.Done()
		}()
	}

	scanner := bufio.NewScanner(hostInput)
	for scanner.Scan() {
		queue <- scanner.Text()
	}
	close(queue)
	wg.Wait()
}

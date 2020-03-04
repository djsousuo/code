/* expects JSON input in the form of { url, method, params }
 * you can get this by piping parseburp output to jq -s '[.[] | {url, method, params}]'
 */
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Targets struct {
	Method string `json:"method"`
	URL    string `json:"url"`
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
			fmt.Println("[*] VULNERABLE (CSRF): " + host + " Parameters: \"" + params + "\"")
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
				fmt.Println("[*] VULNERABLE (CORS): " + host)
			}
		}
	}
}

func main() {
	var err error
	var targets []Targets
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

	queue := make(chan Targets)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			for x := range queue {
				checkHost(x.URL, x.Params, x.Method, client)
			}
			wg.Done()
		}()
	}

	fp, err := ioutil.ReadAll(hostInput)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(fp, &targets)
	if err != nil {
		log.Fatal(err)
	}

	for n := range targets {
		queue <- targets[n]
	}
	close(queue)
	wg.Wait()
}

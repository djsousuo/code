/* expects JSON input in the form of { url, method, params }
 * you can get this by piping parseburp output to jq -s '[.[] | {url, method, params}]'
 */
package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
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

type Payloads struct {
	Attack  string `json:"attack"`
	Payload string `json:"payload"`
}

/*
 * primary HTTP handler
 *
 */
func checkHost(target Targets, payload Payloads, client *http.Client) error {
	req, _ := http.NewRequest(target.Method, target.URL, strings.NewReader(target.Params))
	positive := false

	switch payload.Attack {
	case "cors":
		req.Header.Set("Origin", "evil.com")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp != nil {
			origin := resp.Header.Get("Access-Control-Allow-Origin")
			if origin == "evil.com" || origin == "*" || origin == "*.evil.com" {
				if resp.Header.Get("Access-Control-Allow-Credentials") == "true" {
					positive = true
				}
			}
		}
	case "csrf":
		if target.Method != "POST" {
			return errors.New("Method not POST on CSRF")
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode == 200 {
			positive = true
		}
	case "sqli":
		var start time.Time
		trace := &httptrace.ClientTrace{
			GotFirstResponseByte: func() {
				fmt.Printf("Time since start: %v\n", time.Since(start))
				// this isn't right clearly. placeholder
				if time.Since(start) > 10000 {
					positive = true
				}
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		start = time.Now()
		if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
			return err
		}
	case "xss":
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		if err != nil {
			return err
		}

		if strings.Contains(bodyStr, payload.Payload) {
			positive = true
		}
	}

	if positive == true {
		fmt.Printf("VULNERABLE: (%s - %s) %s %s - Parameters: %s\n", payload.Attack, payload.Payload, target.Method, target.URL, target.Params)
	}

	return nil
}

/*
 * generate valid queries using supplied payloads, for each set of URL/parameters
 *
 */
func generateFuzz(target Targets, payload Payloads) {
	if len(target.Params) > 0 {
		v, err := url.ParseQuery(target.Params)
		if err != nil {
			log.Fatal(err)
		}

		/* make a copy of the map so we can easily permute one key per pass */
		done := make(map[string][]string, len(v))
		for k, v := range v {
			done[k] = v
		}

		/* we need to generate 3 permutations of every value: original, FUZZ, and originalFUZZ */
		q := url.Values{}
		for key := range done {
			for i := 0; i < 2; i++ {
				for k, value := range v {
					if key == k && i == 0 {
						q.Set(k, payload.Payload)
					} else if key == k && i == 1 {
						q.Set(k, string(strings.Join(value, ",")+payload.Payload))
					} else {
						q.Set(k, strings.Join(value, ","))
					}
				}
				if target.Method == "GET" {
					fmt.Printf("%s?%s\n", strings.Split(target.URL, "?")[0], q.Encode())
				}
			}
		}
	}
}

func main() {
	var err error
	var targets []Targets
	var payloads []Payloads
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
			for target := range queue {
				for x := range payloads {
					/*
					 * need to account for 20 (or however many) go routines with X amount of payloads, with multiple variations for each one
					 * throttling will need to be added as well as better go routine management so each one isn't caught up with 500 payloads per URL
					 */
					err := checkHost(target, payloads[x], client)
					if err != nil {
						fmt.Println(err)
					}
				}
			}
			wg.Done()
		}()
	}

	// clean this shit up
	fp, err := ioutil.ReadAll(hostInput)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(fp, &targets)
	if err != nil {
		log.Fatal(err)
	}

	pinput, err := os.Open("payloads.json")
	p, err := ioutil.ReadAll(pinput)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(p, &payloads)
	if err != nil {
		log.Fatal(err)
	}

	for n := range targets {
		queue <- targets[n]
	}
	close(queue)
	wg.Wait()
}

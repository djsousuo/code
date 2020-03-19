/* expects JSON input in the form of { url, method, params }
 * you can get this by piping parseburp output to jq -s '[.[] | {url, method, params}]'
 */
package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
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

type headerFlags []string

type Targets struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Params string `json:"params"`
}

type Payloads struct {
	Attack  string `json:"attack"`
	Payload string `json:"payload"`
}

var Options struct {
	Verbose     bool
	Origin      string
	Host        string
	Concurrency int
	Throttle    int
	Timeout     int
	Headers     headerFlags
	Cookies     headerFlags
	UA          string
}

func (header headerFlags) String() string {
	return strings.Join(header, ",")
}

func (header *headerFlags) Set(str string) error {
	*header = append(*header, str)
	return nil
}

/*
 * primary HTTP handler
 *
 */
func checkHost(target Targets, payload Payloads, client *http.Client) error {
	positive := false
	req, err := http.NewRequest(target.Method, target.URL, strings.NewReader(target.Params))
	if err != nil {
		return err
	}

	if len(Options.Headers) > 0 {
	}

	switch payload.Attack {
	/*
	 * positive: ACAO true with our specified host, ACAC true
	 */
	case "cors":
		req.Header.Set("Origin", Options.Origin)
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return err
		}
		if resp != nil {
			origin := resp.Header.Get("Access-Control-Allow-Origin")
			if origin == Options.Origin || origin == "*" || origin == string("*."+Options.Origin) {
				if resp.Header.Get("Access-Control-Allow-Credentials") == "true" {
					positive = true
				}
			}
		}
	/*
	 * positive: 200 and no CSRF tokens
	 */
	case "csrf":
		if target.Method != "POST" {
			return errors.New("Method not POST on CSRF")
		}
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode == 200 {
			positive = true
		}
	/*
	 * positive: response delay >= whats specified in the injection
	 */
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
	/*
	 * positive: payload reflected in response body
	 */
	case "xss":
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return err
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		bodyStr := string(bodyBytes)

		if strings.Contains(bodyStr, payload.Payload) {
			positive = true
		}
	case "ssrf":
		/* ideally we want some sort of server that spins up and detects these
		 * for the mean time i'll have to rely on a long delay and just weed out the false positives
		 */
	case "lfi":
	case "cmd":
	}

	if positive == true {
		fmt.Printf("[+] VULNERABLE: (%s - %s) %s %s - Parameters: %s\n", payload.Attack, payload.Payload, target.Method, target.URL, target.Params)
	}

	return nil
}

/*
 * generate valid queries using supplied payloads, for each set of URL/parameters
 *
 */
func generateFuzz(target Targets, payload Payloads) ([]Targets, error) {
	var t []Targets
	if len(target.Params) > 0 {
		var js map[string]interface{}
		if json.Unmarshal([]byte(target.Params), &js) == nil {
			// do something with json here
			return nil, errors.New("Invalid JSON parameters")
		}

		v, err := url.ParseQuery(target.Params)
		if err != nil {
			return nil, err
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
				switch target.Method {
				case "GET":
					t = append(t, Targets{target.Method, string(strings.Split(target.URL, "?")[0] + q.Encode()), ""})
				case "POST":
					t = append(t, Targets{target.Method, target.URL, q.Encode()})
				}
			}
		}
	}
	return t, nil
}

func main() {
	var err error
	var targets []Targets
	var payloads []Payloads
	hostInput := os.Stdin

	flag.IntVar(&Options.Concurrency, "c", 20, "Number of concurrent requests")
	flag.IntVar(&Options.Throttle, "d", 2, "Delay between requests (seconds)")
	flag.BoolVar(&Options.Verbose, "v", false, "Verbose mode (default: false)")
	flag.StringVar(&Options.Origin, "o", "evil.com", "Origin")
	flag.StringVar(&Options.UA, "ua", "Mozilla/5.0 (Windows Phone 10.0; Android 6.0.1; Microsoft; RM-1152) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.116 Mobile Safari/537.36 Edge/15.15254", "User Agent")
	flag.Var(&Options.Headers, "h", "Add custom header. Can be specified multiple times")
	flag.Var(&Options.Cookies, "cookie", "Add cookies, seperated by ;")
	flag.IntVar(&Options.Timeout, "timeout", 30, "HTTP timeout (seconds)")
	flag.Parse()

	if len(os.Args) > 1 {
		hostInput, err = os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	}

	var client = &http.Client{
		Timeout: time.Duration(time.Duration(Options.Timeout) * time.Second),
		Transport: &http.Transport{
			MaxIdleConns:    100,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(time.Duration(Options.Timeout) * time.Second),
				KeepAlive: time.Second,
			}).DialContext,
		},
	}

	queue := make(chan Targets)
	var wg sync.WaitGroup

	for i := 0; i < Options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range queue {
				for _, special := range []string{"csrf", "cors"} {
					/* these don't require permutation and will be sent as-is */
					err := checkHost(target, Payloads{special, ""}, client)
					if err != nil && Options.Verbose {
						fmt.Println("[-] Error: ", err)
					}
					time.Sleep(time.Duration(Options.Throttle) * time.Millisecond)
				}

				for x := range payloads {
					/*
					 * need to account for 20 (or however many) go routines with X amount of payloads, with multiple variations for each one
					 * throttling will need to be added as well as better go routine management so each one isn't caught up with 500 payloads per URL
					 *
					 * it will end up being a total of n[params]*2 per payload
					 * with each param being: unmodified, appended with fuzz string, and replaced with fuzz string
					 */
					var currentTarget []Targets
					currentTarget, err = generateFuzz(target, payloads[x])
					if err != nil && Options.Verbose {
						fmt.Println("[-] Error: ", err)
					}

					for y := range currentTarget {
						err := checkHost(currentTarget[y], payloads[x], client)
						if err != nil && Options.Verbose {
							fmt.Println("[-] Error: ", err)
						}
						time.Sleep(time.Duration(Options.Throttle) * time.Millisecond)
					}
				}
			}
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

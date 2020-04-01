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
func checkHost(target Targets, payload Payloads, client *http.Client) (bool, error) {
	positive := false
	req, err := http.NewRequest(target.Method, target.URL, strings.NewReader(target.Params))
	if err != nil {
		return false, err
	}

        req.Header.Set("User-Agent", Options.UA)
	if len(Options.Headers) > 0 {
	}
        //fmt.Printf("URL: %s\n", target.URL)

	switch payload.Attack {
	/*
	 * positive: ACAO true with our specified host, ACAC true
	 */
	case "cors":
		req.Header.Set("Origin", Options.Origin)
		resp, err := client.Do(req)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
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
			return false, errors.New("Method not POST on CSRF")
		}
		resp, err := client.Do(req)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp != nil && resp.StatusCode == 200 {
			positive = true
		}
	/*
	 * positive: response delay >= whats specified in the injection
	 */
        case "ssrf":
		/* ideally we want some sort of server that spins up and detects these
		 * for the mean time i'll have to rely on a long delay and just weed out the false positives
		 */
                fallthrough
	case "sqli":
		var start time.Time
		trace := &httptrace.ClientTrace{
			GotFirstResponseByte: func() {
                                t := time.Since(start)
				if t.Seconds()*10 >= 10 {
                                        //fmt.Printf("[*] Delay - Payload: %s, URL: %s\n", payload.Payload, target.URL)
					positive = true
				}
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		start = time.Now()
		if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
			return false, err
		}
                defer req.Body.Close()
                fallthrough

	/*
	 * positive: payload reflected in response body
	 */
	case "xss":
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			return false, err
		}
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		bodyStr := string(bodyBytes)

		if strings.Contains(bodyStr, "XXSTRINGXX") {
			positive = true
		}
	case "lfi":
                fallthrough
	case "cmd":
                resp, err := client.Do(req)
                if err != nil || resp.StatusCode != 200 {
                        return false, err
                }
                defer resp.Body.Close()

                bodyBytes, err := ioutil.ReadAll(resp.Body)
                if err != nil {
                        return false, err
                }

                bodyStr := string(bodyBytes)
                if strings.Contains(bodyStr, "root:x:") {
                        positive = true
                }

	}

	return positive, nil
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
					t = append(t, Targets{target.Method, string(strings.Split(target.URL, "?")[0] + "?" + q.Encode()), ""})
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
	var endpoint []Targets
	var payloads []Payloads
        var inputFile string
	type job struct {
		t Targets
		p Payloads
	}

	flag.IntVar(&Options.Concurrency, "c", 20, "Number of concurrent requests")
	flag.IntVar(&Options.Throttle, "d", 2, "Delay between requests (milliseconds)")
	flag.BoolVar(&Options.Verbose, "v", false, "Verbose mode (default: false)")
	flag.StringVar(&Options.Origin, "o", "evil.com", "Origin")
        flag.StringVar(&inputFile, "input", "ez.json", "Input file")
	flag.StringVar(&Options.UA, "ua", "Mozilla/5.0 (Windows Phone 10.0; Android 6.0.1; Microsoft; RM-1152) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.116 Mobile Safari/537.36 Edge/15.15254", "User Agent")
	flag.Var(&Options.Headers, "h", "Add custom header. Can be specified multiple times")
	flag.Var(&Options.Cookies, "cookie", "Add cookies, seperated by ;")
	flag.IntVar(&Options.Timeout, "timeout", 30, "HTTP timeout (seconds)")
	flag.Parse()

        hostInput, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
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

	queue := make(chan job)
	var wg sync.WaitGroup

	for i := 0; i < Options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range queue {
				result, err := checkHost(target.t, target.p, client)
				if err != nil && Options.Verbose {
					fmt.Println("[-] Error: ", err)
				} else if err == nil && result == true {
                                        switch target.p.Attack {
                                                case "cors":
                                                        fallthrough
                                                case "csrf":
                                                        fmt.Printf("[+] VULNERABLE: (%s) %s %s\n", strings.ToUpper(target.p.Attack), target.t.Method, target.t.URL)
                                                default:
		                                        fmt.Printf("[+] VULNERABLE: (%s) %s %s - Parameters: %s\n", strings.ToUpper(target.p.Attack), target.t.Method, target.t.URL, target.t.Params)
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

	err = json.Unmarshal(fp, &endpoint)
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

	for n := range endpoint {
                baseURL := strings.Split(endpoint[n].URL, "?")[0]
                if len(baseURL) > 0 && Options.Verbose {
                        fmt.Println("[*] Fuzzing:", baseURL)
                }
		for _, special := range []string{"csrf", "cors"} {
			/* these don't require permutation and will be sent as-is */
                        queue <- job{endpoint[n], Payloads{special, ""}}
			time.Sleep(time.Duration(Options.Throttle) * time.Millisecond)
		}

		for x := range payloads {
			/*
			 * it will end up being a total of params*2 per payload
			 * with each param being: unmodified, appended with fuzz string, and replaced with fuzz string
			 */
                        currentEndpoint, err := generateFuzz(endpoint[n], payloads[x])
			if err != nil && Options.Verbose {
				fmt.Println("[-] Error: ", err)
			}

			for y := range currentEndpoint {
                                queue <- job{currentEndpoint[y], payloads[x]}
				time.Sleep(time.Duration(Options.Throttle) * time.Millisecond)
			}
		}
	}
	close(queue)
	wg.Wait()
}

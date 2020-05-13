package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/olivere/elastic/v7"
	"gopkg.in/elazarl/goproxy.v1"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
)

type vulnType struct {
	Name    string
	Pattern string
}

type logData struct {
	URL      string
	Status   int
	Protocol string
	Method   string
	Params   string
	MIME     string
	Redirect string
	Vuln     []string
	Headers  map[string][]string
}

type config struct {
	sync.Mutex
	Debug     bool
	CheckVuln bool
}

const mapping = `
{
	"settings": {
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings": {
		"proxylog": {
			"properties": {
				"url": {
					"type": "keyword"
				},
				"status": {
					"type": "keyword"
				},
				"protocol": {
					"type": "keyword"
				},
				"method": {
					"type": "keyword"
				},
				"params": {
					"type": "keyword"
				},
				"mime": {
					"type": "keyword"
				},
				"redirect": {
					"type": "text",
					"store": true,
					"fielddata": false
				},
				"vuln": {
					"type": "text",
					"store": true,
					"fielddata": false
				},
				"headers": {
					"type": "text",
					"store": true,
					"fielddata": false
				}
			}
		}
	}
}`

func createRegexp(hosts []byte) (string, int) {
	var whitelist string
	var total int

	for i, host := range strings.Split(string(hosts), "\n") {
		if host == "" {
			continue
		}
		total++
		if i == 0 {
			whitelist = fmt.Sprintf("(^.*\\.?%s", host)
			continue
		}
		whitelist = whitelist + fmt.Sprintf("|^.*\\.?%s", host)
	}
	whitelist = whitelist + fmt.Sprintf(")(\\:.*)?$")
	return whitelist, total
}

func readFile(file string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(file)
	return bytes, err
}

func writeFile(file string, data string) error {
	var err error

	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer f.Close()
		_, err = f.WriteString(data)
	}

	return err
}

func debug(data []byte) {
	var prettyJSON bytes.Buffer

	json.Indent(&prettyJSON, data, "", "\t")
	fmt.Printf("%s\n", string(prettyJSON.Bytes()))
}

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func workerProc(r *http.Response, conf *config) {
	var logReq logData

	vulns := []vulnType{
		{
			Name:    "SSRF",
			Pattern: "http:",
		},
		{
			Name:    "SSRF",
			Pattern: "https:",
		},
		{
			Name:    "SSRF",
			Pattern: "//",
		},
		{
			Name:    "LFI",
			Pattern: "../",
		},
	}

	if r.Request != nil {
		logReq.Status = r.StatusCode
		logReq.MIME = r.Header.Get("Content-Type")
		logReq.Redirect = r.Header.Get("Location")
		if r.Request != nil {
			logReq.URL = r.Request.URL.String()
			logReq.Protocol = r.Request.Proto
			logReq.Method = r.Request.Method
			logReq.Headers = r.Request.Header
		}

		if err := r.Request.ParseForm(); err == nil {
			if params := r.Request.Form.Encode(); params != "" {
				param, err := url.ParseQuery(params)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					if conf.CheckVuln {
						for key, value := range param {
							for each := range value {
								for v := range vulns {
									if strings.Contains(value[each], vulns[v].Pattern) {
										logReq.Vuln = append(logReq.Vuln, fmt.Sprintf("Potential %s (%s) found in parameter %s",
											vulns[v].Name, vulns[v].Pattern, key))
										break
									}
								}
							}
						}
					}
				}
				logReq.Params = params
			}
		}
	}

	data, err := JSONMarshal(logReq)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	if conf.Debug {
		debug(data)
	}

	conf.Lock()
	err = writeFile("out", string(data))
	conf.Unlock()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func elasticProc() {
	ctx := context.Background()
	client, err := elastic.NewClient(elastic.SetURL("http://sto.movsx.dev:9200"), elastic.SetSniff(false))
	if err != nil {
		log.Fatal(err)
	}

	info, code, err := client.Ping("http://sto.movsx.dev:9200").Do(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Elastic returned code %d, version %s\n", code, info.Version.Number)
}

func main() {
	var outputFile string
	var whitelist string
	var concurrent int
	var conf config
	var wg sync.WaitGroup

	flag.StringVar(&outputFile, "o", "data.log", "Output file")
	flag.StringVar(&whitelist, "w", "whitelist", "File with whitelisted domains")
	flag.IntVar(&concurrent, "c", 8, "Number of concurrent processes for parsing")
	flag.BoolVar(&conf.CheckVuln, "C", true, "Check for vulnerabilities in parameters")
	flag.BoolVar(&conf.Debug, "d", false, "Debug mode (print results to stdout)")
	flag.Parse()

	b, err := readFile(whitelist)
	if err != nil {
		log.Fatal(err)
	}

	worker := make(chan *http.Response)
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range worker {
				workerProc(task, &conf)
			}
		}()
	}

	whitelist, total := createRegexp(b)
	fmt.Printf("Added %d hosts to whitelist:\n", total)
	fmt.Printf("%s\n", whitelist)

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.OnResponse(goproxy.ReqHostMatches(regexp.MustCompile(whitelist))).DoFunc(
		func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			worker <- r
			return r
		})

	log.Fatal(http.ListenAndServe("localhost:8081", proxy))
	close(worker)
	wg.Wait()
}

package main

import (
	"bufio"
	"crypto/md5"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Task struct {
	MatchCode      string
	Verbose        bool
	FindFuzz       bool
	FollowRedirect bool
	URL            []string
	Method         string
	Header         []string
	Timeout        int
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomPath() string {
	const possible = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 16)
	for i := range b {
		b[i] = possible[rand.Int63()%int64(len(possible))]
	}

	return "/" + string(b)
}

func findFuzzTarget(url string, task Task, respCode int) {
	if (respCode > 300 && respCode < 400) || respCode == 403 {
		newURL := url + randomPath()
		if fetch(newURL, task) == 404 {
			fmt.Printf("%s\n", url)
		}
	}
}

func matchCodes(match string, status string) bool {
	for _, code := range strings.Split(match, ",") {
		if code == "" {
			continue
		}

		if code == "all" || strings.Contains(status, code) {
			return true
		}
	}

	return false
}

func fetch(host string, task Task) int {
	var client *http.Client

	req, err := http.NewRequest(task.Method, host, nil)
	if err != nil {
		return -1
	}

	if task.FollowRedirect {
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return -1
	}

	if !task.FindFuzz {
		if matchCodes(task.MatchCode, resp.Status) {
			if task.Verbose {
				mime := "UNKNOWN"
				server := "UNKNOWN"
				if len(resp.Header["Content-Type"]) > 0 {
					mime = strings.Split(resp.Header["Content-Type"][0], ";")[0]
				}
				if len(resp.Header["Server"]) > 0 {
					server = resp.Header["Server"][0]
				}
				bodyBytes, _ := ioutil.ReadAll(resp.Body)
				fmt.Printf("%d %s %s %s %s %x ", resp.StatusCode, host, resp.Proto, mime, server, md5.Sum(bodyBytes))
				if len(resp.Header["Location"]) > 0 {
					fmt.Printf("--> %s", resp.Header["Location"][0])
				}
				fmt.Printf("\n")
			} else {
				fmt.Printf("%d %s\n", resp.StatusCode, host)
			}
		}
	}
	resp.Body.Close()
	return resp.StatusCode
}

func main() {
	var concurrency int
	var task Task
	var probe bool
	ports := []string{"81", "300", "591", "593", "832", "981", "1010", "1311", "2082", "2087", "2095", "2096", "2480", "3000", "3128", "3333", "4243", "4567", "4711", "4712", "4993", "5000", "5104", "5108", "5800", "6543", "7000", "7396", "7474", "8000", "8001", "8008", "8014", "8042", "8069", "8080", "8081", "8088", "8090", "8091", "8118", "8123", "8172", "8222", "8243", "8280", "8281", "8333", "8443", "8500", "8834", "8880", "8888", "8983", "9000", "9043", "9060", "9080", "9090", "9091", "9200", "9443", "9800", "9981", "12443", "16080", "18091", "18092", "20720", "28017"}

	flag.StringVar(&task.Method, "X", "GET", "HTTP Method to use (GET/POST)")
	flag.IntVar(&concurrency, "c", 60, "Number of concurrent threads")
	flag.IntVar(&task.Timeout, "t", 5, "Timeout (seconds)")
	flag.StringVar(&task.MatchCode, "mc", "all", "Return only URL's matching HTTP response code")
	flag.BoolVar(&task.FollowRedirect, "f", true, "Follow redirects")
	flag.BoolVar(&task.FindFuzz, "fz", false, "Find fuzzing targets (403/redirect on docroot, 404/200 on /random_file)")
	flag.BoolVar(&task.Verbose, "v", true, "Print verbose information about hosts")
	flag.BoolVar(&probe, "p", false, "httprobe mode")
	flag.Parse()

	timeout := time.Duration(task.Timeout) * time.Second
	http.DefaultTransport = &http.Transport{
		IdleConnTimeout:       time.Second,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ForceAttemptHTTP2:     true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: time.Second,
		}).DialContext,
	}

	tasks := make(chan Task)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				for _, url := range t.URL {
					respCode := fetch(url, t)
					if t.FindFuzz {
						findFuzzTarget(url, t, respCode)
					}
				}
			}
		}()
	}

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		current := input.Text()
		newTask := task
		if probe {
			newTask.URL = append(newTask.URL, "http://" + current)
			newTask.URL = append(newTask.URL, "https://" + current)
			for _, port := range ports {
				newTask.URL = append(newTask.URL, "http://" + current + ":" + port)
				newTask.URL = append(newTask.URL, "https://" + current + ":" + port)
			}
		} else {
			if !strings.HasPrefix(current, "http://") && !strings.HasPrefix(current, "https://") {
				current = "https://" + current
			}
			newTask.URL = append(newTask.URL, current)
		}
		tasks <- newTask
	}
	close(tasks)
	wg.Wait()
}

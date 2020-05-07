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
	URL            string
	Method         string
	Header         []string
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

func findFuzzTarget(task Task, respCode int) {
	if (respCode > 300 && respCode < 400) || respCode == 403 {
		newTask := task
		newTask.URL = newTask.URL + randomPath()
		if fetch(newTask) == 404 {
			fmt.Printf("%s\n", task.URL)
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

func fetch(task Task) int {
	var client *http.Client

	req, err := http.NewRequest(task.Method, task.URL, nil)
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
	defer resp.Body.Close()

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
				fmt.Printf("%d %s %s %s %x ", resp.StatusCode, task.URL, mime, server, md5.Sum(bodyBytes))
				if len(resp.Header["Location"]) > 0 {
					fmt.Printf("--> %s", resp.Header["Location"][0])
				}
				fmt.Printf("\n")
			} else {
				fmt.Printf("%d %s\n", resp.StatusCode, task.URL)
			}
		}
	}
	return resp.StatusCode
}

func main() {
	var concurrency int
	var timeout int
	var task Task

	flag.StringVar(&task.Method, "X", "GET", "HTTP Method to use (GET/POST)")
	flag.IntVar(&concurrency, "c", 20, "Number of concurrent requests (default: 20)")
	flag.IntVar(&timeout, "t", 10, "Timeout (seconds)")
	flag.StringVar(&task.MatchCode, "mc", "all", "Return only URL's matching HTTP response code")
	flag.BoolVar(&task.FollowRedirect, "f", true, "Follow redirects")
	flag.BoolVar(&task.FindFuzz, "fz", false, "Find fuzzing targets (403/redirect on docroot, 404/200 on /random_file)")
	flag.BoolVar(&task.Verbose, "v", false, "Print verbose information about hosts")
	flag.Parse()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	http.DefaultTransport.(*http.Transport).DialContext = (&net.Dialer{
		Timeout:   time.Duration(timeout) * time.Second,
		KeepAlive: time.Second,
	}).DialContext

	tasks := make(chan Task)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				respCode := fetch(t)
				if t.FindFuzz {
					findFuzzTarget(t, respCode)
				}
			}
		}()
	}

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		current := input.Text()
		if !strings.HasPrefix(current, "http://") && !strings.HasPrefix(current, "https://") {
			current = "https://" + current
		}
		task.URL = current

		tasks <- task
	}
	close(tasks)
	wg.Wait()
}

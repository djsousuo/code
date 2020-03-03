package main

import (
	"bufio"
	"crypto/tls"
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

func checkHost(host string, client *http.Client) {
	originCheck := true
	req, err := http.NewRequest("GET", host, nil)
	if err != nil {
		return
	}

	if originCheck == true {
		req.Header.Set("Origin", "evil.com")
	}
	/*
	        req.Header.Set("PHPSESSID", "gm3lb9d2fi28tvna4sncteb043")
		req.Header.Set("Host", "movsx.dev")
	*/

	resp, err := client.Do(req)

	if resp != nil {
		if resp.StatusCode == 405 {
			fmt.Println("[*] Bad request: " + host)
		}
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		if strings.Contains(bodyStr, "pageReady") {
			fmt.Println("[*] Vuln: " + host)
		}
		origin := resp.Header.Get("Access-Control-Allow-Origin")
		if origin == "evil.com" || origin == "*" || origin == "*.evil.com" {
			fmt.Println("[*] Host matched Origin header: " + host)
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

	timeout := time.Duration(1000000 * 10000)
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

	hosts := make(chan string)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			for host := range hosts {
				checkHost(host, client)
			}
			wg.Done()
		}()
	}

	scanner := bufio.NewScanner(hostInput)
	for scanner.Scan() {
		hosts <- scanner.Text()
	}

	close(hosts)
	wg.Wait()
}

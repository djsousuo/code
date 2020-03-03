package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Fingerprints struct {
	Service     string   `json:"service"`
	Cname       []string `json:"cname"`
	Fingerprint []string `json:"fingerprint"`
	Nxdomain    bool     `json:"nxdomain"`
}

func dnsCname(host string, fp []Fingerprints) (string, bool) {
	cname := ""
	msg := new(dns.Msg)
	msg.SetQuestion(string(strings.Split(host, "://")[1])+".", dns.TypeCNAME)
	reply, err := dns.Exchange(msg, "4.2.2.4:53")

	if err != nil {
		return "[-] DNS Lookup", false
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.CNAME); ok {
			cname = t.Target
		}
	}

	for n := range fp {
		for x := range fp[n].Cname {
			if strings.Contains(cname, fp[n].Cname[x]) {
				return cname, true
			}
		}
	}

	return "[-] No matches found", false
}

func dnsNx(host string, cname string, fp []Fingerprints) bool {
	name := strings.Split(host, "://")[1]
	_, err := net.LookupHost(name)

	if err != nil {
		for n := range fp {
			if fp[n].Nxdomain == true {
				for x := range fp[n].Cname {
					if strings.Contains(cname, fp[n].Cname[x]) && strings.Contains(err.Error(), "no such host") {
						return true
					}
				}
			}
		}
	}

	return false
}

func checkHost(host string, cname string, client *http.Client, fp []Fingerprints) bool {
	req, err := http.NewRequest("GET", host, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "User-Agent: Mozilla/5.0 (Linux; Android 4.0.4; Galaxy Nexus Build/IMM76B) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.133 Mobile Safari/535.19")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	if resp != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		for n := range fp {
			for x := range fp[n].Cname {
				if strings.Contains(cname, fp[n].Cname[x]) {
					for z := range fp[n].Fingerprint {
						if strings.Contains(bodyStr, fp[n].Fingerprint[z]) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func readConfig(file string) (data []Fingerprints) {
	fp, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(fp, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func main() {
	var err error
	config := "./config.json"
	hostInput := os.Stdin

	if len(os.Args) > 1 {
		hostInput, err = os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	}

	timeout := time.Duration(100000 * 100000)
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

	var fpItems []Fingerprints
	fpItems = readConfig(config)

	hosts := make(chan string)
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			for host := range hosts {
				cname, found := dnsCname(host, fpItems)
				if found {
					if dnsNx(host, cname, fpItems) {
						fmt.Println("[*] VULNERABLE (NXDOMAIN): " + host + " (" + cname + ")")
						continue
					}

					if checkHost(host, cname, client, fpItems) {
						fmt.Println("[*] VULNERABLE: " + host + " (" + cname + ")")
						continue
					}

				}
			}
			wg.Done()
		}()
	}

	scanner := bufio.NewScanner(hostInput)
	for scanner.Scan() {
		hosts <- scanner.Text()
	}

	wg.Wait()
	close(hosts)
}

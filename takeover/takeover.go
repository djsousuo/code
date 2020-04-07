package main

import (
	"bufio"
	"bytes"
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

type Bot struct {
	Username string `json:"username"`
	Webhook  string `json:"webhook"`
}

type Message struct {
	Username string `json:"username"`
	Content  string `json:"content"`
}

var Results struct {
        Output []string
        Len int
}

func discordMsg(msg string, config Bot, client *http.Client) error {
        format := "```" + msg + "```"
	m := Message{config.Username,
		format}

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", config.Webhook, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func dnsCname(host string, fp []Fingerprints) (string, bool, int) {
	cname := ""
	msg := new(dns.Msg)
	msg.SetQuestion(string(strings.Split(host, "://")[1])+".", dns.TypeCNAME)
	reply, err := dns.Exchange(msg, "4.2.2.4:53")

	if err != nil {
		return "", false, -1
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.CNAME); ok {
			cname = t.Target
		}
	}

	for n := range fp {
		for x := range fp[n].Cname {
			if strings.Contains(cname, fp[n].Cname[x]) {
				return cname, true, n
			}
		}
	}

	return "", false, -1
}

func dnsNx(host string, cname string) bool {
	name := strings.Split(host, "://")[1]
	_, err := net.LookupHost(name)

	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return true
		}
	}

	return false
}

func checkHost(host string, cname string, client *http.Client, fp []string) bool {
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
		resp.Body.Close()
		bodyStr := string(bodyBytes)
		for i := range fp {
			if strings.Contains(bodyStr, fp[i]) {
				return true
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
        var start time.Time
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

	var botConfig Bot
	fp, err := ioutil.ReadFile("bot.json")
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(fp, &botConfig)
	if err != nil {
		log.Fatal(err)
	}

	hosts := make(chan string)
	var wg sync.WaitGroup
        var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for host := range hosts {
                                /* send to Discord webhook every minute, or every 16 positive results */
                                mu.Lock()
                                t := time.Since(start)
                                if Results.Len >= 16 || (Results.Len >= 1 && t.Seconds() >= 60) {
				        err := discordMsg(strings.Join(Results.Output, "\n"), botConfig, client)
				        if err != nil {
					        fmt.Println(err)
				        }
                                        Results.Len = 0
                                        Results.Output = nil
                                        start = time.Now()
                                }
                                mu.Unlock()
				cname, found, index := dnsCname(host, fpItems)

				if found {
                                        /* NXDOMAIN takeover */
					if fpItems[index].Nxdomain && dnsNx(host, cname) {
						str := string("[*] " + strings.ToUpper(fpItems[index].Service) + " NXDOMAIN: " + host + " CNAME: " + cname)
						fmt.Println(str)
                                                mu.Lock()
                                                Results.Len += 1
                                                Results.Output = append(Results.Output, str)
                                                mu.Unlock()
						continue
					}

                                        /* traditional CNAME takeover with website fingerprint */
					if checkHost(host, cname, client, fpItems[index].Fingerprint) {
						str := string("[*] " + strings.ToUpper(fpItems[index].Service) + " " + host + " CNAME: " + cname)
						fmt.Println(str)
                                                mu.Lock()
                                                Results.Len += 1
                                                Results.Output = append(Results.Output, str)
                                                mu.Unlock()
						continue
					}
				}
			}
			wg.Done()
		}()
	}

        start = time.Now()
	scanner := bufio.NewScanner(hostInput)
	for scanner.Scan() {
		current := scanner.Text()
		if !strings.HasPrefix(current, "http://") && !strings.HasPrefix(current, "https://") {
			current = "https://" + current
		}

		hosts <- current
	}

	close(hosts)
	wg.Wait()
}

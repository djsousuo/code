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

type Message struct {
	Username string `json:"username"`
	Message  string `json:"content"`
}

type State struct {
	sync.Mutex
	Start   time.Time
	Client  *http.Client
	Output  *[]string
	Entries *int
}

var Config struct {
	Discord struct {
		Username string  `json:"username"`
		Webhook  string  `json:"webhook"`
		Timeout  float64 `json:"timeout"`
		MaxLen   int     `json:"maxentries"`
	}
	NS          string
	UA          string
	Timeout     int
	Retries     int
	Verbose     bool
	Concurrency int
}
var fpItems []Fingerprints

func discordMsg(msg string, state *State) error {
	format := "```" + msg + "```"
	m := Message{Config.Discord.Username,
		format}

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", Config.Discord.Webhook, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := state.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func discordTimer(state *State) {
	state.Lock()
	l := state.Len()
	t := time.Since(state.Start)
	if l >= Config.Discord.MaxLen || (l >= 1 && t.Seconds() >= Config.Discord.Timeout) {
		if err := discordMsg(strings.Join(*state.Output, "\n"), state); err != nil {
			fmt.Println(err)
		}
		state.Clear()
	}
	state.Unlock()
}

func absoluteHost(host string) string {
	if strings.Contains(host, "://") {
		return strings.Split(host, "://")[1] + "."
	} else {
		return host + "."
	}
}

func dnsCNAME(host string) (string, error) {
	var cname string
	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeA)
	reply, err := dns.Exchange(msg, Config.NS)

	if err != nil {
		return "", err
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.CNAME); ok {
			cname = t.Target
		}
	}

	return cname, nil
}

func dnsNS(host string) []string {
	var ns []string
	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeNS)
	reply, err := dns.Exchange(msg, Config.NS)
	if err != nil {
		return nil
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.NS); ok {
			ns = append(ns, t.Ns)
		}
	}

	return ns
}

func dnsA(host string, ns string) (string, bool) {
	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeA)
	reply, err := dns.Exchange(msg, ns+":53")
	if err != nil {
		return "", false
	}

	if reply.Rcode == dns.RcodeServerFailure || reply.Rcode == dns.RcodeRefused {
		fmt.Println("[*] Found AWS NXDOMAIN with host: " + host + ", NS: " + ns)
		return ns, true
	}
	return "", false
}

func dnsNX(host string) bool {
	if _, err := net.LookupHost(host); err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return true
		}
	}
	return false
}

func matchAWS(host []string) {
}

func matchFingerprint(cname string) (bool, int) {
	if cname == "" {
		return false, -1
	}

	for n := range fpItems {
		for x := range fpItems[n].Cname {
			if strings.Contains(cname, fpItems[n].Cname[x]) {
				return true, n
			}
		}
	}

	return false, -1
}

func checkHost(host string, state *State) {
	aHost := absoluteHost(host)
	nx := dnsNX(strings.Split(host, "://")[1])

	cname, err := dnsCNAME(aHost)

	/* edge case: route53 with SERVFAIL/REFUSED at awsdns */
	if nx && err == nil {
		ns := dnsNS(aHost)
		if ns != nil {
			for i := range ns {
				if strings.Contains(ns[i], "awsdns") {
					if cname != "" {
						dnsA(cname, ns[i])
					} else {
						dnsA(aHost, ns[i])
					}
				}
			}
		}
	}

	found, index := matchFingerprint(cname)
	if found {
		/* NXDOMAIN takeover */
		if fpItems[index].Nxdomain && nx {
			str := fmt.Sprintf("[*] %s NXDOMAIN: %s CNAME: %s", strings.ToUpper(fpItems[index].Service), host, cname)
			fmt.Println(str)
			state.Update(str)
			return
		}

		/* traditional CNAME takeover with website fingerprint */
		if verifyFingerprint(host, cname, fpItems[index].Fingerprint, state) {
			str := fmt.Sprintf("[*] %s %s CNAME: %s", strings.ToUpper(fpItems[index].Service), host, cname)
			fmt.Println(str)
			state.Update(str)
			return
		}
	}
}

func verifyFingerprint(host string, cname string, fp []string, state *State) bool {
	req, err := http.NewRequest("GET", host, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "User-Agent: Mozilla/5.0 (Linux; Android 4.0.4; Galaxy Nexus Build/IMM76B) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.133 Mobile Safari/535.19")

	resp, err := state.Client.Do(req)
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

func loadJSON(file string, v interface{}) {
	fp, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(fp, &v); err != nil {
		log.Fatal(err)
	}
}

func (s *State) Len() int {
	return *s.Entries
}

func (s *State) Clear() {
	*s.Entries = 0
	*s.Output = nil
	s.Start = time.Now()
}

func (s *State) Update(str string) {
	s.Lock()
	*s.Entries += 1
	*s.Output = append(*s.Output, str)
	s.Unlock()
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

	timeout := 15 * time.Second
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

	var state = &State{
		Entries: new(int),
		Output:  new([]string),
		Start:   time.Now(),
		Client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}

	loadJSON("config.json", &fpItems)
	loadJSON("bot.json", &Config.Discord)
	Config.NS = "4.2.2.4:53"
	hosts := make(chan string)
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range hosts {
				//discordTimer(state)
				checkHost(host, state)
			}
		}()
	}

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

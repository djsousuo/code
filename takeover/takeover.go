package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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
	} `json:"discord"`
	Resolvers   string `json:"resolvers"`
	UA          string `json:"user_agent"`
	Timeout     int    `json:"timeout"`
	Retries     int    `json:"retries"`
	Concurrency int    `json:"concurrency"`
	NSList      []string
	Rand        *rand.Rand
}
var fpItems []Fingerprints

func discordMsg(msg string, state *State) error {
	type message struct {
		Username string `json:"username"`
		Message  string `json:"content"`
	}

	if len(msg) < 1 {
		return nil
	}

	format := "```" + msg + "```"
	m := message{
		Username: Config.Discord.Username,
		Message:  format,
	}

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
	_ = resp.Body.Close()

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
	}

	return host + "."
}

func matchFingerprint(cname string) *Fingerprints {
	for n := range fpItems {
		for x := range fpItems[n].Cname {
			if strings.Contains(cname, fpItems[n].Cname[x]) {
				return &fpItems[n]
			}
		}
	}

	return nil
}

func checkHost(host string, state *State) {
	var cname, _ = dnsCNAME(absoluteHost(host), Config.Retries)
	if cname == "" {
		return
	}

	if found := matchFingerprint(cname); found != nil {
		nx := false
		if answer, err := dnsA(cname, Config.Retries); answer == nil && err == nil {
			nx = true
		}

		if (nx && found.Nxdomain) || (!nx && !found.Nxdomain && verifyFingerprint(host, found.Fingerprint, state)) {
			str := fmt.Sprintf("[*] %s %s CNAME: %s", strings.ToUpper(found.Service), host, cname)
			fmt.Println(str)
			state.Update(str)
		}
	}
}

func verifyFingerprint(host string, fp []string, state *State) bool {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}

	req, err := http.NewRequest("GET", host, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", Config.UA)

	resp, err := state.Client.Do(req)
	if err != nil {
		return false
	}

	if resp != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		bodyStr := strings.ToLower(string(bodyBytes))
		for i := range fp {
			if strings.Contains(bodyStr, strings.ToLower(fp[i])) {
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

	loadJSON("fingerprints.json", &fpItems)
	loadJSON("config.json", &Config)

	resolverInput, err := os.Open(Config.Resolvers)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(resolverInput)
	for scanner.Scan() {
		Config.NSList = append(Config.NSList, scanner.Text())
	}
	resolverInput.Close()

	Config.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	timeout := time.Duration(Config.Timeout) * time.Second
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

	hosts := make(chan string, Config.Concurrency * 20)
	var wg sync.WaitGroup

	for i := 0; i < Config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range hosts {
				if Config.Discord.Webhook != "" {
					discordTimer(state)
				}
				checkHost(host, state)
			}
		}()
	}

	scanner = bufio.NewScanner(hostInput)
	for scanner.Scan() {
		hosts <- scanner.Text()
	}

	close(hosts)
	wg.Wait()
}

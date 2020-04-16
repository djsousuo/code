package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"errors"
)

var Opt struct {
	NS      string
	Reverse bool
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomSub() string {
	const possible = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 16)
	for i := range b {
		b[i] = possible[rand.Int63()%int64(len(possible))]
	}

	return string(b)
}

func splitHost(host string) (domain string, err error) {
	var d string

	s := strings.Split(host, ".")
	if len(s) < 3 {
		return host, errors.New("splitHost: already at TLD")
	}

	s = s[1:]
	for i := range s {
		d = d + "." + s[i]
	}
	return d, nil
}

func resolve(host string) bool {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Duration(15 * time.Second),
			}
			return d.DialContext(ctx, "udp", string(Opt.NS+":53"))
		},
	}

	if _, err := resolver.LookupHost(context.Background(), host); err != nil {
		return false
	}

	return true
}

func main() {
	var inWait sync.WaitGroup
	var outWait sync.WaitGroup
	var mu sync.Mutex

	flag.StringVar(&Opt.NS, "n", "4.2.2.4", "Nameserver to use for lookups")
	flag.BoolVar(&Opt.Reverse, "r", false, "Reverse mode (only show hosts with no wildcards)")
	flag.Parse()

	allDomains := make(map[string]bool)
	input := bufio.NewScanner(os.Stdin)
	in := make(chan string)
	out := make(chan string)

	outWait.Add(1)
	go func() {
		for o := range out {
			fmt.Println(o)
		}
		outWait.Done()
	}()

	for i := 0; i < 30; i++ {
		inWait.Add(1)
		go func() {
			for h := range in {
				domain, err := splitHost(h)

				mu.Lock()
				_, found := allDomains[domain]
				allDomains[domain] = true
				mu.Unlock()

				if err != nil || found {
					if Opt.Reverse {
						fmt.Println(h)
					}
					continue
				}

				result := resolve(randomSub() + domain)

				if !result && Opt.Reverse {
					out <- h
				} else if result && !Opt.Reverse {
					out <- domain
				}
			}
			inWait.Done()
		}()
	}

	for input.Scan() {
		in <- input.Text()
	}
	close(in)
	inWait.Wait()
	close(out)
	outWait.Wait()
}

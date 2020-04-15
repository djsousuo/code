package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomSub() string {
	const possible = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 12)
	for i := range b {
		b[i] = possible[rand.Int63()%int64(len(possible))]
	}

	return string(b)
}

func splitHost(host string) (domain string, try bool) {
	var d string

	s := strings.Split(host, ".")
	if len(s) < 3 {
		return host, false
	}

	s = s[1:]
	for i := range s {
		d = d + "." + s[i]
	}
	return d, true
}

func resolve(host string) bool {
	addrs, err := net.LookupHost(host)
	if err != nil || addrs == nil {
		return false
	}

	return true
}

func main() {
	allDomains := make(map[string]bool)
	input := bufio.NewScanner(os.Stdin)

	for input.Scan() {
		domain, try := splitHost(input.Text())
		if !try {
			continue
		}

		if _, ok := allDomains[domain]; ok {
			continue
		}
		allDomains[domain] = true

		for i := 0; i < 5; i++ {
			if found := resolve(randomSub() + domain); found {
				fmt.Println(domain)
				break
			}
		}
	}
}

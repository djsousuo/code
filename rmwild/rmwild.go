package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

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
	var inWait sync.WaitGroup
	var outWait sync.WaitGroup

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
				if found := resolve(randomSub() + h); found {
					out <- h
				}
			}
			inWait.Done()
		}()
	}

	for input.Scan() {
		domain, try := splitHost(input.Text())
		_, ok := allDomains[domain]
		if !try || ok {
			continue
		}

		allDomains[domain] = true
		in <- domain
	}
	close(in)
	inWait.Wait()
	close(out)
	outWait.Wait()
}

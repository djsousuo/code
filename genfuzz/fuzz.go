package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

func createFuzz(method string, baseUrl string, params string) {
	if len(params) > 0 {

		v, err := url.ParseQuery(params)
		if err != nil {
			log.Fatal(err)
		}

		/* make a copy of the map so we can easily permute one key per pass */
		done := make(map[string][]string, len(v))
		for k, v := range v {
			done[k] = v
		}

		/* we need to generate 3 permutations of every value: original, FUZZ, and originalFUZZ */
		q := url.Values{}
		for key := range done {
			for i := 0; i < 2; i++ {
				for k, value := range v {
					if key == k && i == 0 {
						q.Set(k, "FUZZ")
					} else if key == k && i == 1 {
						q.Set(k, string(strings.Join(value, ",")+"FUZZ"))
					} else {
						q.Set(k, strings.Join(value, ","))
					}
				}
				if method == "GET" {
					fmt.Printf("%s?%s\n", strings.Split(baseUrl, "?")[0], q.Encode())
				}
			}
		}
	}
}

func main() {
}

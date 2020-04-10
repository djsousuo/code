package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type Data struct {
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

func main() {
	var find []string
	var data Data

	if len(os.Args) < 3 {
		fmt.Println("Usage: ./extract <wildcard list> <fdns dataset>")
		return
	}

        wild, err := os.Open(os.Args[1])
        if err != nil {
                log.Fatal(err)
        }
        defer wild.Close()
        scanner := bufio.NewScanner(wild)
        for scanner.Scan() {
                find = append(find, scanner.Text())
        }

	in, err := os.Open(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	reader, err := gzip.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	scanner = bufio.NewScanner(reader)

	for scanner.Scan() {
		err := json.Unmarshal(scanner.Bytes(), &data)
		if err != nil {
			log.Fatal(err)
		}

		for host := range find {
			if strings.HasSuffix(data.Name, string("." + find[host])) {
				fmt.Fprintf(out, data.Name+"\n")
			}
		}

	}
}

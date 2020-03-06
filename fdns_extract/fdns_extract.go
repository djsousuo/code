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
	find := []string{"fastly", "github.io", "pantheonsite.io", "wordpress.com", "teamwork.com", "helpjuice.com", "helpscoutdocs", "ghost.io", "feedpress", "statuspage",
		"uservoice", "surge.sh", "intercom.help", "bitbucket.io", "webflow.com", "wishpond", "aftership", "aha.io", "tictail", "bcvp0rtal", "brightcove", "gallery.video",
		"bigcartel", "createsend", "acquia", "simplebooklet", "gr8.com", "vendecommerce", "myjetbrains", "azure", "windows.net", "cloudapp.net", "visualstudio", "zendesk",
		"readme.io", "apigee", "smugmug", "fly", "tilda", "frontify", "landingi", "canny", "ngrok", "hubspot", "smartling"}

	if len(os.Args) < 2 {
		fmt.Println("Usage: ./extract <fdns dataset>")
		return
	}

	in, err := os.Open(os.Args[1])
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

	scanner := bufio.NewScanner(reader)
	var data Data

	for scanner.Scan() {
		err := json.Unmarshal(scanner.Bytes(), &data)
		if err != nil {
			log.Fatal(err)
		}

		for host := range find {
			if strings.Contains(data.Value, find[host]) {
				fmt.Fprintf(out, data.Name+"\n")
			}
		}

	}
}

package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
	"os"
)

type Entry struct {
	Result []struct {
		Input struct {
			Fuzz string `json:"FUZZ"`
		} `json:"input"`
		Position int    `json:"position"`
		Status   int    `json:"status"`
		Length   int    `json:"length"`
		Words    int    `json:"words"`
		Lines    int    `json:"lines"`
		Redirect string `json:"resultlocation"`
		URL      string `json:"url"`
	} `json:"results"`
}

func main() {
	var entries []Entry

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <ffuf_output.json>\n", os.Args[0])
		return
	}

	for i := 1; i < len(os.Args); i++ {
		fp, err := ioutil.ReadFile(os.Args[i])

		if err != nil {
			log.Fatal(err)
		}

		var entry Entry
		if err = json.Unmarshal(fp, &entry); err != nil {
			log.Fatal(err)
		}
		entries = append(entries, entry)
	}

	for i := range entries {
		for r := range entries[i].Result {
			var redirect bool
			c := &color.Color{}
			current := entries[i].Result[r]
			status := current.Status

			switch {
			case status == 200:
				c = color.New(color.FgGreen)
			case status >= 300 && status < 400:
				c = color.New(color.FgBlue)
				redirect = true
			case status > 400 && status < 600:
				c = color.New(color.FgRed)
			case status == 404:
				c = color.New(color.FgYellow)
			default:
				c = color.New(color.FgWhite)
			}

			c.Printf("%d\t%d\t%s", current.Status, current.Length, current.URL)
			if redirect && len(current.Redirect) > 0 {
				c.Printf(" --> %s", current.Redirect)
			}
			c.Printf("\n")
		}
	}
}

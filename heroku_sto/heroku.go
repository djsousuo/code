package main

import (
	"fmt"
	"github.com/bgentry/heroku-go"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No domain specified, exiting...")
		os.Exit(1)
	}
	host := os.Args[1]

	client := heroku.Client{Username: "user", Password: "password"}
	app, err := client.AppCreate(nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Created app %s\n", app.Name)

	domain, err := client.DomainCreate(app.Name, host)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully created domain: %s\n", domain.Hostname)
}

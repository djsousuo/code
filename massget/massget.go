package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
)

func ErrorMsg(url string, err error) {
	fmt.Printf("%s ERROR\n")
}

func Worker(url string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		ErrorMsg(url, err)
		return
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		ErrorMsg(url, err)
		return
	}

	resp.Body.Close()
	fmt.Println(url, resp.StatusCode, resp.Header["Content-Type"], resp.Header["Content-Length"], resp.Header["Server"])
}

func main() {
	var method string
	var verbose bool

	flag.StringVar(&method, "method", "GET", "HTTP Method to use (GET/POST)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	if verbose {
		fmt.Println("Using HTTP Method:", method)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		Worker(input.Text())
	}
}

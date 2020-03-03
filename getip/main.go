package main

import (
        "fmt"
        "regexp"
        "io/ioutil"
        "net/http"
)

func main() {
        URL := "https://www.ipchicken.com"

        req, err := http.NewRequest(http.MethodGet, URL, nil)
        if err != nil {
                panic(err)
        }

        client := http.DefaultClient
        resp, err := client.Do(req)
        if err != nil {
                panic(err)
        }

        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if (err != nil) {
                panic(err)
        }

        re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
        fmt.Println("IP address:", string(re.Find(body)))
}

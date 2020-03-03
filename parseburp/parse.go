package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"
)

type BurpItems struct {
	XMLName   xml.Name   `xml:"items"`
	BurpItems []BurpItem `xml:"item"`
}

type BurpItem struct {
	XMLName xml.Name `xml:"item"`
	Time    string   `xml:"time"`
	URL     string   `xml:"url"`
	Host    string   `xml:"host"`
	Port    int      `xml:"port"`
	Proto   string   `xml:"protocol"`
	Method  string   `xml:"method"`
	Path    string   `xml:"path"`
	Ext     string   `xml:"extension"`
	Req     string   `xml:"request"`
	Status  int      `xml:"status"`
	Length  int      `xml:"responselength"`
	Mime    string   `xml:"mimetype"`
	Resp    string   `xml:"response"`
	Comment string   `xml:"comment"`
}

type Logs struct {
	Time    string `json:"time"`
	URL     string `json:"url"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Proto   string `json:"protocol"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Req     string `json:"request"`
	Status  int    `json:"status"`
	Length  int    `json:"responselength"`
	Mime    string `json:"mimetype"`
	Resp    string `json:"response"`
	Headers string `json:"headers"`
	Params  string `json:"params"`
}

func decodeURL(data string) string {
	tmp, err := url.QueryUnescape(data)
	if err != nil {
		return "Error: decodeURL()"
	}
	return tmp
}

func decodeBase64(data string) string {
	if len(data) < 1 {
		return "NULL"
	}

	b64, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "Error: decodeBase64()"
	}

	if !unicode.IsPrint(rune(b64[0])) {
		return "<Binary Data>"
	}

	return string(b64)
}

/* separate an HTTP request into (headers, body) */
func parseRequest(request string) (string, string) {
	var headers []string

	scanner := bufio.NewReader(strings.NewReader(request))

	req, _ := http.ReadRequest(scanner)
	for name, head := range req.Header {
		for _, h := range head {
			headers = append(headers, fmt.Sprintf("%v: %v", name, h))
		}
	}

	err := req.ParseForm()
	if err != nil {
		fmt.Println("Error: ParseForm()")
		return "error", "error"
	}

	return strings.Join(headers, "\n"), req.Form.Encode()
}

func main() {
	var err error
	sessionFile := os.Stdin

	if len(os.Args) > 1 {
		sessionFile, err = os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	}

	byteValue, _ := ioutil.ReadAll(sessionFile)

	var itemsXML BurpItems
	err = xml.Unmarshal(byteValue, &itemsXML)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(itemsXML.BurpItems); i++ {
		cur := itemsXML.BurpItems[i]

		if len(cur.Mime) < 1 {
			cur.Mime = "UNKNOWN"
		} else {
			cur.Mime = strings.ToUpper(cur.Mime)
		}

		headers, params := parseRequest(decodeBase64(cur.Req))
		m := Logs{cur.Time,
			decodeURL(cur.URL),
			cur.Host,
			cur.Port,
			cur.Proto,
			cur.Method,
			cur.Path,
			decodeBase64(cur.Req),
			cur.Status,
			cur.Length,
			cur.Mime,
			decodeBase64(cur.Resp),
			headers,
			params}

		data, err := json.Marshal(m)
		if err != nil {
			log.Fatal(err)
		}

		var prettyJSON bytes.Buffer
		json.Indent(&prettyJSON, data, "", "\t")
		fmt.Println(string(prettyJSON.Bytes()))
	}
}

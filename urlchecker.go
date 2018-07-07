package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"time"
)

// Node - every link found stored as a node
type Node struct {
	Link         string
	RedirectURL  string
	StatusCode   int
	Headers      http.Header
	Dump         string
	ResponseTime int
}

// Link -
type Link struct {
	EndPoint string `json:"url"`
}

// urlCheck -
// This function is for URL checking and will return a single Node
func urlCheck(url string) (Node, error) {

	var data Node
	var t0, t1, t2, t3, t4, t5, t6 time.Time

	timeout := time.Duration(30 * time.Second)
	c := http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest("GET", url, nil)
	//	resp, err := client.Get(url)
	if err != nil {
		log.Print(err)
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { t0 = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
		ConnectStart: func(_, _ string) {
			if t1.IsZero() {
				// connecting to IP
				t1 = time.Now()
			}
		},
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Fatalf("unable to connect to host %v: %v", addr, err)
			}
			t2 = time.Now()
		},
		GotConn:              func(_ httptrace.GotConnInfo) { t3 = time.Now() },
		GotFirstResponseByte: func() { t4 = time.Now() },
		TLSHandshakeStart:    func() { t5 = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { t6 = time.Now() },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	resp, err := c.Do(req)
	resp.Body.Close()
	t7 := time.Now() // after read body

	data.Link = url
	d := t7.Sub(t0).Round(time.Millisecond)
	data.ResponseTime = int(d) / 1000000
	data.StatusCode = resp.StatusCode
	if resp.Request.URL.String() != url {
		data.RedirectURL = resp.Request.URL.String()
	}
	if data.StatusCode == 404 {
		// Save a copy of this request for debugging.
		requestDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			fmt.Println(err)
		}
		data.Dump = string(requestDump)
		data.Headers = resp.Header
	}

	tags := map[string]string{
		"url":          data.Link,
		"responseCode": string(data.StatusCode),
	}

	fields := map[string]interface{}{
		"response_time": data.ResponseTime,
		"response_code": data.StatusCode,
	}
	statHandler(tags, fields, data.Link)
	return data, err

}

// runChecker - This just loops over a slice of strings
func runChecker(l string) Node {
	n, e := urlCheck(l)
	if e != nil {
		fmt.Printf("Got Error: %s when fetching: %s", e, l)
	}
	return n
}

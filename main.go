package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// ***** Structs For Web Service *******
// This is a very simple checkip and geoip information service
type key int

// Link -
type Link struct {
	EndPoint string `json:"url"`
}

// Node - every link found stored as a node
type Node struct {
	Link         string
	RedirectURL  string
	StatusCode   int
	Headers      http.Header
	Dump         string
	ResponseTime int
}

// ***** END STRUCTS *********

const (
	requestIDKey key = 0
)

var (
	listenAddr string
	healthy    int32
	body       []byte
)

func main() {
	// Default to port 3000 on localhost
	// You can pass the --listen-addr flag, need to include port
	flag.StringVar(&listenAddr, "listen-addr", ":3000", "server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      tracing(nextRequestID)(logging(logger)(routes())),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Listen for CTRL+C or kill and start shutting down the app without
	// disconnecting people by not taking any new requests. ("Graceful Shutdown")
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	logger.Println("Server stopped")

}

// routes -
// Setup all your routes simple mux router
// Put new handler routes here
func routes() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", checkHandler)
	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/ping", pingHandler)
	return router
}

// ****** HANDLERS HERE ********

// checkHandler -
// executes the url checking and parsing the json payload
func checkHandler(w http.ResponseWriter, r *http.Request) {
	var lk Link
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	temp, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(temp, &lk)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	data := runChecker(lk.EndPoint)
	output, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if data.StatusCode == 404 {
		fmt.Println("Got a 404, this is where I'd send a email!")
		//	triggerMessage("Hello from LinkFN!\n"+"Link Checked: "+data.Link+"\n"+string(output), data.Link)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(output)
}

// batchCheckHandler

// pingHandler -
// Simple health check.
func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "{\"status\":\"pong!\"}")
}

// forceTextHandler -
// Prevent Content-Type sniffing
func forceTextHandler(w http.ResponseWriter, r *http.Request) {
	// https://stackoverflow.com/questions/18337630/what-is-x-content-type-options-nosniff
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "{\"status\":\"ok\"}")
}

// healthHandler -
// Report server status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&healthy) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, "{\"status\":\"bad\"}")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "{\"status\":\"ok\"}")

}

// ****** END HANDLERS HERE *******

// ****** START FUNC's ******

// logging just a simple logging handler
// this generates a basic access log entry
func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// tracing for debuging a access log entry to a given request
func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
	fmt.Println(d)
	data.ResponseTime = int(d)
	//fmt.Printf("Took %d ms\n", data.ResponseTime)
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

	//data.ResponseTime = int(result.ContentTransfer(time.Now()) / time.Millisecond)
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

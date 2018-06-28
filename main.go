package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"gopkg.in/mailgun/mailgun-go.v1"
)

// **** MAILGUN SETTINGS ****
// TODO: Move to ENV's in the OS or Container
// mailgun domain names
var yourDomain string = os.Getenv("MAILGUN_DOMAIN") //"mg.makerdev.nl" // e.g. mg.yourcompany.com

// starts with "key-"
var privateAPIKey string = os.Getenv("MAILGUN_PRIV_API_KEY") //"41ba7c4ab2eeae72e230e99ce31d445f-e44cc7c1-b556d011"

// starts with "pubkey-"
var publicValidationKey string = os.Getenv("MAILGUN_PUBLIC_VALID_KEY") //"pubkey-8e185f8d9740bd85d4e41d0bf6b7e510"

// Send messages to
var sendTo string = os.Getenv("MAIL_RECPT") //"richard.genthner@makerbot.com"

// Who the messages are from
var replyTo string = os.Getenv("MAIL_REPLY_TO") //"no-reply@makerbot.com"

// **** END MAILGUN SETTINGS *****

// ***** Structs For Web Service *******
// This is a very simple checkip and geoip information service
type key int

// Link -
type Link struct {
	EndPoint string `json:"url"`
}

// Node - every link found stored as a node
type Node struct {
	Link        string
	RedirectURL string
	StatusCode  int
	Headers     http.Header
	Dump        string
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
		triggerMessage("Hello from LinkFN!\n"+"Link Checked: "+data.Link+"\n"+string(output), data.Link)
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

	timeout := time.Duration(30 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		log.Print(err)
	}
	data.Link = url
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

// triggerMessage -
// This used for sending the output via email and building Mailgun object
func triggerMessage(message string, link string) {
	// Create an instance of the Mailgun Client
	mg := mailgun.NewMailgun(yourDomain, privateAPIKey, publicValidationKey)

	sender := replyTo
	subject := "404 Detected: " + link
	body := message
	recipient := sendTo

	sendMessage(mg, sender, subject, body, recipient)
}

// sendMessage -
// Mailgun message sender
func sendMessage(mg mailgun.Mailgun, sender, subject, body, recipient string) {
	message := mg.NewMessage(sender, subject, body, recipient)
	resp, id, err := mg.Send(message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
}

// ***** END FUNC's HERE ******

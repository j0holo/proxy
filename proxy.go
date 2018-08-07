package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type apiRequest struct {
	URL    string              `json:"url"`
	Header map[string][]string `json:"header"`
}

type response struct {
	URL        string              `json:"url"`
	StatusCode int                 `json:"status_code"`
	Header     map[string][]string `json:"header"`
	Body       string              `json:"body"`
	Status     string              `json:"status"`
}

var (
	// A channel that keeps track of the requests per second.
	counter = make(chan time.Duration, 4096)

	apiKey = os.Getenv("proxy_api_key")
	port   = os.Getenv("proxy_port")
)

func main() {
	f, err := os.OpenFile("proxy.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)

	// Listen to signals of the OS
	var signals = make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGINT)
	go handleSignal(signals)

	proxyHandler := http.HandlerFunc(proxyHandler)
	statsHandler := http.HandlerFunc(statsHandler)
	http.Handle("/", authenticate(proxyHandler))
	http.HandleFunc("/check", checkHandler)
	http.Handle("/stats", authenticate(statsHandler))

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	srv := &http.Server{
		Handler:      nil,
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	log.Printf("Server is running on 0.0.0.0:%s...", port)
	// Start requests per second counter.
	go performanceCounter()

	serverCrt, exists := os.LookupEnv("server_crt")
	serverKey, exists := os.LookupEnv("server_key")

	if exists != true {
		serverCrt = "cert.pem"
	}

	if exists != true {
		serverKey = "key.pem"
	}
	log.Fatal(srv.ListenAndServeTLS(serverCrt, serverKey))
}

// proxyHandler processes every proxy request.
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Only POST requests are supported.
	if r.Method != http.MethodPost {
		internalServerError(w, fmt.Errorf("Method not allowed %s, %s", r.Method, r.RemoteAddr))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		proxyErrorHandler(w, "", 0, err)
		return
	}
	defer r.Body.Close()

	var request apiRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		// If the parsing went wrong can't use request.URL.
		proxyErrorHandler(w, "", 0, err)
		return
	}

	response, err := proxyRequest(request)
	if err != nil {
		proxyErrorHandler(w, request.URL, 0, err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// statsHandler shows statistics about the proxy.
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("No statistics yet."))
}

// checkHandler returns a 200 status code.
func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

func authenticate(h http.Handler) http.Handler {
	start := time.Now()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != apiKey {
			log.Println("Unauthorized access:", r.RemoteAddr)
			internalServerError(w, fmt.Errorf("Unauthorized access: %s", r.RemoteAddr))
			return
		}
		h.ServeHTTP(w, r)
		counter <- time.Since(start)
	})
}

// proxyRequest fetches the given URL and returns a response struct.
func proxyRequest(a apiRequest) (response, error) {
	c := http.Client{Timeout: time.Duration(10 * time.Second)}

	req, err := http.NewRequest(http.MethodGet, a.URL, nil)
	if err != nil {
		return response{URL: a.URL, Status: err.Error()}, err
	}

	// Add all the given headers to the request.
	for key, values := range a.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := c.Do(req)
	if err != nil {
		return response{URL: a.URL, Status: err.Error()}, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response{URL: a.URL, Status: err.Error()}, err
	}

	answer := response{
		URL:        a.URL,
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Header:     make(map[string][]string),
		Status:     "",
	}

	for field, value := range resp.Header {
		answer.Header[field] = value
	}

	return answer, nil
}

func proxyErrorHandler(w http.ResponseWriter, url string, statusCode int, err error) {
	errorResponse, marshalError := json.Marshal(response{URL: url, Status: err.Error(), StatusCode: statusCode})
	if marshalError != nil {
		log.Printf("Failed to create ErrorResponse: %v", marshalError)
	}

	http.Error(w, string(errorResponse), http.StatusOK)
}

func internalServerError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, fmt.Sprintf(`{ "status": "%s"}`, http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError)
}

func handleSignal(sigChan chan os.Signal) {
	sig := <-sigChan

	log.Printf("Caught signal: %+v.", sig)
	log.Println("Server is exiting.")
	os.Exit(0)
}

// performanceCounter reads the counter channel and stores the result every minute.
func performanceCounter() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		var nReq float64          // number of requests.
		var sumTime time.Duration // sum of the time of every request.
		for _ = range ticker.C {
			if l := len(counter); l == 0 {
				fmt.Println("0 requests have been made.")
			} else {
				for i := 0; i < l; i++ {
					sumTime += <-counter
					nReq++
				}

				avgTime := sumTime.Seconds() / nReq
				fmt.Printf("Number of requests: %f\nAverage time: %f\n", nReq, avgTime)
				nReq = 0
				sumTime = 0
			}
		}
	}()

}

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
}

const (
	apiKey = "oqV8oCApJZvdE5+25fSxrbSakIcnzFnGi/bxuPeeYhcWeMfUbj3Q+sTDtLDMWsfxBNYXFAyyic6ZT7IhW/+nqzt7hJw2A1DYCxU+3xsiYlYQ96Z6Ks1VM9p7fYor1FElftOeU06hNNu41tUXAVf2P2N/FbeQrsLUHIPkld2zaXOl2nBNJB5vTdc1enuz/MicGAzhpeE/dOED0lkMY5aYPVCoWjgkayOwb6J4kRObojbGtsFMDGdk7zCLqBApoE26nxNYCfvN/8OX4JQxdkbDC06adrWsvkSi1YGPyWEe7UAQrPvVmhDpHQ6GthSWar59ybZEf2rbIR90dwXb9F6Ndg=="
	port   = 4443
)

var (
	// A channel that keeps track of the requests per second.
	counter = make(chan time.Duration, 4096)
)

func main() {
	f, err := os.OpenFile("proxy.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)

	proxyHandler := http.HandlerFunc(proxyHandler)
	statsHandler := http.HandlerFunc(statsHandler)
	http.Handle("/", authenticate(proxyHandler))
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
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	log.Printf("Server is running on 0.0.0.0:%d...", port)
	// Start requests per second counter.
	go performanceCounter()

	log.Fatal(srv.ListenAndServeTLS("server.crt", "server.key"))
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
		internalServerError(w, err)
		return
	}
	defer r.Body.Close()

	var request apiRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		internalServerError(w, err)
		return
	}

	response, err := proxyRequest(request)
	if err != nil {
		internalServerError(w, err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// statsHandler shows statistics about the proxy
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("No statistics yet."))
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
	c := http.Client{}
	req, err := http.NewRequest(http.MethodGet, a.URL, nil)
	if err != nil {
		return response{}, err
	}

	// Add all the given headers to the request.
	log.Println(a)
	for key, values := range a.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := c.Do(req)
	if err != nil {
		return response{}, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response{}, err
	}

	answer := response{
		URL:        a.URL,
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Header:     make(map[string][]string),
	}

	for field, value := range resp.Header {
		answer.Header[field] = value
	}

	return answer, nil
}

func internalServerError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

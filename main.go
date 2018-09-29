package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	apiKey = os.Getenv("proxy_api_key")
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

	r := newHandler()
	tlsCfg := newTLSConfig()
	srv := newTLSServer(tlsCfg, r)

	log.Fatal(srv.ListenAndServeTLS(getTLSCertificates()))
}

func newTLSServer(tlsCfg *tls.Config, mux *http.ServeMux) *http.Server {
	port := os.Getenv("proxy_port")
	if port == "" {
		port = "4443"
	}

	srv := &http.Server{
		Handler:      mux,
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	return srv
}

func newHandler() *http.ServeMux {
	r := http.NewServeMux()
	proxyHandler := http.HandlerFunc(proxyHandler)
	r.Handle("/", authenticate(proxyHandler))
	r.HandleFunc("/check", checkHandler)

	return r
}

func newTLSConfig() *tls.Config {
	return &tls.Config{
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

func getTLSCertificates() (string, string) {
	serverCrt, exists := os.LookupEnv("server_crt")
	serverKey, exists := os.LookupEnv("server_key")

	if exists != true {
		serverCrt = "cert.pem"
	}

	if exists != true {
		serverKey = "key.pem"
	}
	return serverCrt, serverKey
}

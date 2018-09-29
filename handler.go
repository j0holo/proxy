package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

// proxyHandler processes every proxy request.
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Only POST requests are supported.
	if r.Method != http.MethodPost {
		internalServerError(w, fmt.Errorf("Method not allowed %s, %s", r.Method, r.RemoteAddr))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// If the parsing went wrong can't use request.URL.
		// Maybe we should ignore the request instead of answering it?
		proxyErrorHandler(w, "", 0, err)
		return
	}
	defer r.Body.Close()

	var request apiRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		// If the parsing went wrong can't use request.URL.
		// Maybe we should ignore the request instead of answering it?
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

// checkHandler returns a 200 status code to tell HaProxy that it is healthy to accept requests.
func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

// authenticate is a middleware handler to authenticate requests to a given http.Handler.
func authenticate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != apiKey {
			internalServerError(w, fmt.Errorf("Unauthorized access"))
			return
		}
		h.ServeHTTP(w, r)
	})
}

// proxyErrorHandler returns an error response to the extractor that the website didn't
// behave as expected and that no usefull information could be harvested from the website.
// The URL and status code from the website will be returned with the accompanying error message.
func proxyErrorHandler(w http.ResponseWriter, url string, statusCode int, err error) {
	errorResponse, marshalError := json.Marshal(response{URL: url, Status: err.Error(), StatusCode: statusCode})
	if marshalError != nil {
		log.Printf("Failed to create ErrorResponse: %v", marshalError)
	}

	http.Error(w, string(errorResponse), http.StatusOK)
}

package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

// proxyRequest fetches the given URL and returns a response struct.
func proxyRequest(a apiRequest) (response, error) {
	resp, err := getURL(a.URL, a.Header)
	if err != nil {
		return buildResponse(a.URL, 0, "", nil, err.Error()), err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return buildResponse(a.URL, resp.StatusCode, "", nil, err.Error()), err
	}

	r := buildResponse(a.URL, resp.StatusCode, string(body), resp.Header, "")
	return r, nil
}

// buildResponse returns a response for a extractor that will later be converted to a JSON object.
func buildResponse(URL string, statusCode int, body string, headers map[string][]string, status string) response {
	return response{
		URL:        URL,
		StatusCode: statusCode,
		Body:       body,
		Header:     headers,
		Status:     status,
	}
}

// getURL returns a *http.Response of the given URL.
func getURL(URL string, headers map[string][]string) (*http.Response, error) {
	c := http.Client{Timeout: time.Duration(10 * time.Second)}

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	// Add all the given headers to the request.
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	return c.Do(req)
}

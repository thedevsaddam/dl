package downloader

import "net/http"

// HTTPClient interface
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response
	Do(req *http.Request) (*http.Response, error)
}

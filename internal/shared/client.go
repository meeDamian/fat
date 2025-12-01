package shared

import (
	"net/http"
	"time"
)

// HTTPClient is an interface that matches http.Client's Do method
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultTransport is a shared transport with optimized connection pooling settings
var defaultTransport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
	IdleConnTimeout:     90 * time.Second,
}

// NewHTTPClient creates a new HTTP client with the shared transport and specified timeout
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: defaultTransport,
	}
}

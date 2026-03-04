package net

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/waffo-com/waffo-go/errors"
)

// Default timeout values in milliseconds
const (
	DefaultConnectTimeout = 10000 // 10 seconds
	DefaultReadTimeout    = 30000 // 30 seconds
)

// DefaultHttpTransport is the default HTTP transport implementation using net/http.
// It enforces TLS 1.2+ and provides connection pooling.
type DefaultHttpTransport struct {
	client *http.Client
}

// NewDefaultHttpTransport creates a new DefaultHttpTransport with default settings.
func NewDefaultHttpTransport() *DefaultHttpTransport {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12, // Enforce TLS 1.2+
		},
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(DefaultConnectTimeout) * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &DefaultHttpTransport{
		client: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(DefaultReadTimeout) * time.Millisecond,
		},
	}
}

// NewDefaultHttpTransportWithTimeouts creates a new DefaultHttpTransport with custom timeouts.
func NewDefaultHttpTransportWithTimeouts(connectTimeout, readTimeout int64) *DefaultHttpTransport {
	if connectTimeout <= 0 {
		connectTimeout = DefaultConnectTimeout
	}
	if readTimeout <= 0 {
		readTimeout = DefaultReadTimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12, // Enforce TLS 1.2+
		},
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(connectTimeout) * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &DefaultHttpTransport{
		client: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(readTimeout) * time.Millisecond,
		},
	}
}

// Send sends an HTTP request and returns the response.
func (t *DefaultHttpTransport) Send(ctx context.Context, req *HttpRequest) (*HttpResponse, error) {
	// Create HTTP request
	var bodyReader io.Reader
	if req.Body != nil {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to create HTTP request", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Use custom client if timeouts are specified
	client := t.client
	if req.ConnectTimeout > 0 || req.ReadTimeout > 0 {
		// Create a new client with custom timeouts for this request
		connectTimeout := req.ConnectTimeout
		readTimeout := req.ReadTimeout
		if connectTimeout <= 0 {
			connectTimeout = DefaultConnectTimeout
		}
		if readTimeout <= 0 {
			readTimeout = DefaultReadTimeout
		}

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(connectTimeout) * time.Millisecond,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		client = &http.Client{
			Transport: transport,
			Timeout:   time.Duration(readTimeout) * time.Millisecond,
		}
	}

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		// Check if it's a network error
		if isNetworkError(err) {
			return nil, errors.NewNetworkError("network error, payment status unknown", err)
		}
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "HTTP request failed", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to read response body", err)
	}

	// Extract headers
	headers := make(map[string]string)
	for key := range resp.Header {
		headers[key] = resp.Header.Get(key)
	}

	return NewHttpResponse(resp.StatusCode, headers, body), nil
}

// isNetworkError checks if the error is a network-related error.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Check for connection refused, DNS errors, etc.
	if _, ok := err.(*net.OpError); ok {
		return true
	}

	return false
}

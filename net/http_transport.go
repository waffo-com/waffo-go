// Package net provides HTTP transport layer for the Waffo SDK.
package net

import "context"

// HttpTransport is the interface for HTTP transport implementations.
// This allows users to provide custom HTTP transport (e.g., for proxies, custom TLS config).
type HttpTransport interface {
	// Send sends an HTTP request and returns the response.
	// The context can be used for cancellation and timeout control.
	Send(ctx context.Context, req *HttpRequest) (*HttpResponse, error)
}

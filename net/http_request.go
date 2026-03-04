package net

// HttpRequest represents an HTTP request.
type HttpRequest struct {
	// URL is the full URL to send the request to.
	URL string

	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// Headers are the HTTP headers to send.
	Headers map[string]string

	// Body is the request body (for POST, PUT, etc.).
	Body []byte

	// ConnectTimeout is the TCP connection timeout in milliseconds.
	// If 0, the default timeout is used.
	ConnectTimeout int64

	// ReadTimeout is the response read timeout in milliseconds.
	// If 0, the default timeout is used.
	ReadTimeout int64
}

// NewHttpRequest creates a new HTTP request with default values.
func NewHttpRequest(method, url string) *HttpRequest {
	return &HttpRequest{
		URL:     url,
		Method:  method,
		Headers: make(map[string]string),
	}
}

// SetHeader sets a header on the request.
func (r *HttpRequest) SetHeader(key, value string) *HttpRequest {
	r.Headers[key] = value
	return r
}

// SetBody sets the request body.
func (r *HttpRequest) SetBody(body []byte) *HttpRequest {
	r.Body = body
	return r
}

// SetTimeout sets the connect and read timeouts in milliseconds.
func (r *HttpRequest) SetTimeout(connectTimeout, readTimeout int64) *HttpRequest {
	r.ConnectTimeout = connectTimeout
	r.ReadTimeout = readTimeout
	return r
}

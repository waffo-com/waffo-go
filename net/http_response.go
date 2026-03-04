package net

// HttpResponse represents an HTTP response.
type HttpResponse struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// Headers are the response headers.
	Headers map[string]string

	// Body is the response body.
	Body []byte
}

// NewHttpResponse creates a new HTTP response.
func NewHttpResponse(statusCode int, headers map[string]string, body []byte) *HttpResponse {
	return &HttpResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}
}

// GetHeader returns the value of a header (case-insensitive lookup).
func (r *HttpResponse) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	// HTTP headers are case-insensitive
	for k, v := range r.Headers {
		if equalFold(k, key) {
			return v
		}
	}
	return ""
}

// IsSuccess returns true if the status code is 2xx.
func (r *HttpResponse) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// equalFold is a simple case-insensitive string comparison.
func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c1 := s[i]
		c2 := t[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

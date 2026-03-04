package config

// RequestOptions contains per-request options that can override default config.
type RequestOptions struct {
	// ConnectTimeout is the TCP connection timeout in milliseconds.
	// If 0, the default from WaffoConfig is used.
	ConnectTimeout int64

	// ReadTimeout is the response read timeout in milliseconds.
	// If 0, the default from WaffoConfig is used.
	ReadTimeout int64

	// Headers are additional headers to include in the request.
	Headers map[string]string
}

// NewRequestOptions creates a new RequestOptions with default values.
func NewRequestOptions() *RequestOptions {
	return &RequestOptions{
		Headers: make(map[string]string),
	}
}

// WithConnectTimeout sets the connect timeout.
func (o *RequestOptions) WithConnectTimeout(timeout int64) *RequestOptions {
	o.ConnectTimeout = timeout
	return o
}

// WithReadTimeout sets the read timeout.
func (o *RequestOptions) WithReadTimeout(timeout int64) *RequestOptions {
	o.ReadTimeout = timeout
	return o
}

// WithHeader adds a header to the request.
func (o *RequestOptions) WithHeader(key, value string) *RequestOptions {
	if o.Headers == nil {
		o.Headers = make(map[string]string)
	}
	o.Headers[key] = value
	return o
}

// Merge merges two RequestOptions, with the other taking precedence.
func (o *RequestOptions) Merge(other *RequestOptions) *RequestOptions {
	if other == nil {
		return o
	}

	result := &RequestOptions{
		ConnectTimeout: o.ConnectTimeout,
		ReadTimeout:    o.ReadTimeout,
		Headers:        make(map[string]string),
	}

	// Copy headers from o
	for k, v := range o.Headers {
		result.Headers[k] = v
	}

	// Override with other
	if other.ConnectTimeout > 0 {
		result.ConnectTimeout = other.ConnectTimeout
	}
	if other.ReadTimeout > 0 {
		result.ReadTimeout = other.ReadTimeout
	}
	for k, v := range other.Headers {
		result.Headers[k] = v
	}

	return result
}

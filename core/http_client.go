// Package core provides core functionality for the Waffo SDK.
package core

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/errors"
	"github.com/waffo-com/waffo-go/net"
	"github.com/waffo-com/waffo-go/utils"
)

// HTTP Headers
const (
	HeaderContentType  = "Content-Type"
	HeaderAPIKey       = "X-API-KEY"
	HeaderSignature    = "X-SIGNATURE"
	HeaderAPIVersion   = "X-API-VERSION"
	HeaderSDKVersion   = "X-SDK-VERSION"
	ContentTypeJSON    = "application/json"
	APIVersion         = "1.0.0"
)

// WaffoHttpClient is the HTTP client for making API requests.
type WaffoHttpClient struct {
	config    *config.WaffoConfig
	transport net.HttpTransport
}

// NewWaffoHttpClient creates a new WaffoHttpClient.
func NewWaffoHttpClient(cfg *config.WaffoConfig) *WaffoHttpClient {
	transport := cfg.CustomTransport
	if transport == nil {
		transport = net.NewDefaultHttpTransportWithTimeouts(cfg.ConnectTimeout, cfg.ReadTimeout)
	}

	return &WaffoHttpClient{
		config:    cfg,
		transport: transport,
	}
}

// RawResponse represents the raw API response format.
type RawResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"msg,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Post sends a POST request to the specified path.
// It handles serialization, signing, and response verification.
func (c *WaffoHttpClient) Post(ctx context.Context, path string, request interface{}, opts *config.RequestOptions) (*RawResponse, error) {
	// 1. Inject merchantId if configured
	c.injectMerchantID(request)

	// 1.5. Inject requestedAt timestamp if applicable
	c.injectRequestedAt(request)

	// 2. Serialize request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeSerializationFailed, "failed to serialize request", err)
	}

	// 3. Sign the request body
	signature, err := utils.Sign(string(requestBody), c.config.PrivateKey)
	if err != nil {
		return nil, err
	}

	// 4. Build HTTP request
	url := c.config.GetBaseURL() + path
	httpReq := net.NewHttpRequest("POST", url).
		SetHeader(HeaderContentType, ContentTypeJSON).
		SetHeader(HeaderAPIKey, c.config.APIKey).
		SetHeader(HeaderSignature, signature).
		SetHeader(HeaderAPIVersion, APIVersion).
		SetHeader(HeaderSDKVersion, c.config.GetSDKVersion()).
		SetBody(requestBody)

	// Apply request options
	if opts != nil {
		if opts.ConnectTimeout > 0 {
			httpReq.ConnectTimeout = opts.ConnectTimeout
		}
		if opts.ReadTimeout > 0 {
			httpReq.ReadTimeout = opts.ReadTimeout
		}
		for k, v := range opts.Headers {
			httpReq.SetHeader(k, v)
		}
	}

	// 5. Send request
	httpResp, err := c.transport.Send(ctx, httpReq)
	if err != nil {
		// Error is already wrapped by transport
		return nil, err
	}

	// 6. Verify response signature (if present)
	responseSignature := httpResp.GetHeader(HeaderSignature)
	if responseSignature != "" {
		if !utils.Verify(string(httpResp.Body), responseSignature, c.config.WaffoPublicKey) {
			return nil, errors.NewWaffoError(errors.CodeVerificationFailed, "response signature verification failed")
		}
	}

	// 7. Parse response
	var rawResp RawResponse
	if err := json.Unmarshal(httpResp.Body, &rawResp); err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to parse response", err)
	}

	// 8. Check for E0001 (unknown status)
	if rawResp.Code == "E0001" {
		return nil, errors.NewWaffoUnknownStatusError(errors.CodeUnknownStatus, "unknown status from server")
	}

	return &rawResp, nil
}

// injectMerchantID injects the merchantId into the request if configured.
func (c *WaffoHttpClient) injectMerchantID(request interface{}) {
	if c.config.MerchantID == "" {
		return
	}

	// Use reflection to inject merchantId
	v := reflect.ValueOf(request)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	// Check for merchantInfo.merchantId
	merchantInfo := v.FieldByName("MerchantInfo")
	if merchantInfo.IsValid() {
		// Handle pointer type (*MerchantInfo)
		if merchantInfo.Kind() == reflect.Ptr {
			if merchantInfo.IsNil() {
				if merchantInfo.CanSet() {
					merchantInfo.Set(reflect.New(merchantInfo.Type().Elem()))
				} else {
					return
				}
			}
			merchantInfo = merchantInfo.Elem()
		}
		if merchantInfo.Kind() == reflect.Struct {
			merchantID := merchantInfo.FieldByName("MerchantID")
			if merchantID.IsValid() && merchantID.CanSet() && merchantID.Kind() == reflect.String {
				if merchantID.String() == "" {
					merchantID.SetString(c.config.MerchantID)
				}
			}
		}
	}

	// Check for top-level merchantId
	merchantID := v.FieldByName("MerchantID")
	if merchantID.IsValid() && merchantID.CanSet() && merchantID.Kind() == reflect.String {
		if merchantID.String() == "" {
			merchantID.SetString(c.config.MerchantID)
		}
	}
}

// injectRequestedAt injects a timestamp into OrderRequestedAt or RequestedAt fields if empty.
func (c *WaffoHttpClient) injectRequestedAt(request interface{}) {
	v := reflect.ValueOf(request)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	now := utils.GetCurrentTimestamp()
	// Order series: OrderRequestedAt
	if f := v.FieldByName("OrderRequestedAt"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String && f.String() == "" {
		f.SetString(now)
	}
	// Subscription series: RequestedAt
	if f := v.FieldByName("RequestedAt"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String && f.String() == "" {
		f.SetString(now)
	}
}

// PostWithResponse is a generic helper that posts and unmarshals the data field.
func PostWithResponse[T any](c *WaffoHttpClient, ctx context.Context, path string, request interface{}, opts *config.RequestOptions) (*ApiResponse[T], error) {
	rawResp, err := c.Post(ctx, path, request, opts)
	if err != nil {
		return nil, err
	}

	// Check if success
	if rawResp.Code != "0" {
		return &ApiResponse[T]{
			Code:    rawResp.Code,
			Message: rawResp.Message,
		}, nil
	}

	// Unmarshal data
	var data T
	if len(rawResp.Data) > 0 {
		if err := json.Unmarshal(rawResp.Data, &data); err != nil {
			return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to parse response data", err)
		}
	}

	return &ApiResponse[T]{
		Code:    rawResp.Code,
		Message: rawResp.Message,
		Data:    &data,
	}, nil
}

// ApiResponse represents the API response wrapper.
type ApiResponse[T any] struct {
	Code    string `json:"code"`
	Message string `json:"msg,omitempty"`
	Data    *T     `json:"data,omitempty"`
}

// IsSuccess returns true if the response code is "0".
func (r *ApiResponse[T]) IsSuccess() bool {
	return r.Code == "0"
}

// GetCode returns the response code.
func (r *ApiResponse[T]) GetCode() string {
	return r.Code
}

// GetMessage returns the response message.
func (r *ApiResponse[T]) GetMessage() string {
	return r.Message
}

// GetData returns the response data.
func (r *ApiResponse[T]) GetData() *T {
	return r.Data
}

// Error returns an error response with the given code and message.
func Error[T any](code, message string) *ApiResponse[T] {
	return &ApiResponse[T]{
		Code:    code,
		Message: message,
	}
}

// Success returns a success response with the given data.
func Success[T any](data *T) *ApiResponse[T] {
	return &ApiResponse[T]{
		Code: "0",
		Data: data,
	}
}

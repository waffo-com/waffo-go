// Package types provides type definitions for the Waffo SDK.
package types

// MerchantInfo represents merchant information in requests.
type MerchantInfo struct {
	MerchantID string `json:"merchantId,omitempty"`
}

// ExtraParams allows passing additional dynamic parameters.
type ExtraParams map[string]interface{}

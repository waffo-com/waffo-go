// Package merchant provides merchant-related types.
package merchant

import "github.com/waffo-com/waffo-go/types"

// InquiryMerchantConfigParams represents the parameters for querying merchant config.
type InquiryMerchantConfigParams struct {
	MerchantInfo types.MerchantInfo `json:"merchantInfo"`
}

// InquiryMerchantConfigData represents the response data for merchant config inquiry.
type InquiryMerchantConfigData struct {
	MerchantID          string   `json:"merchantId,omitempty"`
	MerchantName        string   `json:"merchantName,omitempty"`
	SupportedCurrencies []string `json:"supportedCurrencies,omitempty"`
	SupportedMethods    []string `json:"supportedMethods,omitempty"`
}

// InquiryPayMethodConfigParams represents the parameters for querying payment method config.
type InquiryPayMethodConfigParams struct {
	MerchantInfo  types.MerchantInfo `json:"merchantInfo"`
	PaymentMethod string             `json:"paymentMethod,omitempty"`
	Currency      string             `json:"currency,omitempty"`
}

// InquiryPayMethodConfigData represents the response data for payment method config inquiry.
type InquiryPayMethodConfigData struct {
	PaymentMethods []PaymentMethodConfig `json:"paymentMethods,omitempty"`
}

// PaymentMethodConfig represents a payment method configuration.
type PaymentMethodConfig struct {
	PaymentMethod    string   `json:"paymentMethod,omitempty"`
	PaymentMethodName string  `json:"paymentMethodName,omitempty"`
	Currencies       []string `json:"currencies,omitempty"`
	MinAmount        string   `json:"minAmount,omitempty"`
	MaxAmount        string   `json:"maxAmount,omitempty"`
}

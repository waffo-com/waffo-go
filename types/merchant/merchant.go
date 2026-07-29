// Package merchant provides merchant-related types.
package merchant

import "github.com/waffo-com/waffo-go/types"

// InquiryMerchantConfigParams represents the parameters for querying merchant config.
type InquiryMerchantConfigParams struct {
	MerchantID  string            `json:"merchantId"`
	ExtraParams types.ExtraParams `json:"extraParams,omitempty"`
}

// InquiryMerchantConfigData represents the response data for merchant config inquiry.
//
// The limit fields are returned as JSON strings keyed by currency, for example
// `{"PHP":"100000","IDR":"2000000"}`, and are empty when no limit is set.
type InquiryMerchantConfigData struct {
	MerchantID          string `json:"merchantId,omitempty"`
	TotalDailyLimit     string `json:"totalDailyLimit,omitempty"`
	RemainingDailyLimit string `json:"remainingDailyLimit,omitempty"`
	TransactionLimit    string `json:"transactionLimit,omitempty"`
}

// InquiryPayMethodConfigParams represents the parameters for querying payment method config.
type InquiryPayMethodConfigParams struct {
	MerchantID  string            `json:"merchantId"`
	ExtraParams types.ExtraParams `json:"extraParams,omitempty"`
}

// InquiryPayMethodConfigData represents the response data for payment method config inquiry.
type InquiryPayMethodConfigData struct {
	MerchantID       string            `json:"merchantId,omitempty"`
	PayMethodDetails []PayMethodDetail `json:"payMethodDetails,omitempty"`
}

// PayMethodDetail represents a payment method detail from config inquiry.
type PayMethodDetail struct {
	ProductName              string                 `json:"productName,omitempty"`
	PayMethodName            string                 `json:"payMethodName,omitempty"`
	Country                  string                 `json:"country,omitempty"`
	CurrentStatus            string                 `json:"currentStatus,omitempty"`
	FixedMaintenanceRules    []FixedMaintenanceRule `json:"fixedMaintenanceRules,omitempty"`
	FixedMaintenanceTimezone string                 `json:"fixedMaintenanceTimezone,omitempty"`
	PayMethodType            string                 `json:"payMethodType,omitempty"`
	Currency                 string                 `json:"currency,omitempty"`
	OrderAmountMin           string                 `json:"orderAmountMin,omitempty"`
	OrderAmountMax           string                 `json:"orderAmountMax,omitempty"`
	RefundAllowed            *int                   `json:"refundAllowed,omitempty"`
}

// FixedMaintenanceRule represents a fixed maintenance rule.
type FixedMaintenanceRule struct {
	StartRule string `json:"startRule,omitempty"`
	EndRule   string `json:"endRule,omitempty"`
}

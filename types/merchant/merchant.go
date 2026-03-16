// Package merchant provides merchant-related types.
package merchant

import "github.com/waffo-com/waffo-go/types"

// InquiryMerchantConfigParams represents the parameters for querying merchant config.
type InquiryMerchantConfigParams struct {
	MerchantID  string            `json:"merchantId"`
	ExtraParams types.ExtraParams `json:"extraParams,omitempty"`
}

// InquiryMerchantConfigData represents the response data for merchant config inquiry.
type InquiryMerchantConfigData struct {
	MerchantID          string            `json:"merchantId,omitempty"`
	TotalDailyLimit     map[string]string `json:"totalDailyLimit,omitempty"`
	RemainingDailyLimit map[string]string `json:"remainingDailyLimit,omitempty"`
	TransactionLimit    map[string]string `json:"transactionLimit,omitempty"`
}

// InquiryPayMethodConfigParams represents the parameters for querying payment method config.
type InquiryPayMethodConfigParams struct {
	MerchantID    string            `json:"merchantId"`
	ExtraParams   types.ExtraParams `json:"extraParams,omitempty"`
}

// InquiryPayMethodConfigData represents the response data for payment method config inquiry.
type InquiryPayMethodConfigData struct {
	MerchantID       string            `json:"merchantId,omitempty"`
	PayMethodDetails []PayMethodDetail `json:"payMethodDetails,omitempty"`
}

// PayMethodDetail represents a payment method detail from config inquiry.
type PayMethodDetail struct {
	ProductName              string                `json:"productName,omitempty"`
	PayMethodName            string                `json:"payMethodName,omitempty"`
	Country                  string                `json:"country,omitempty"`
	CurrentStatus            string                `json:"currentStatus,omitempty"`
	FixedMaintenanceRules    []FixedMaintenanceRule `json:"fixedMaintenanceRules,omitempty"`
	FixedMaintenanceTimezone string                `json:"fixedMaintenanceTimezone,omitempty"`
}

// FixedMaintenanceRule represents a fixed maintenance rule.
type FixedMaintenanceRule struct {
	StartRule string `json:"startRule,omitempty"`
	EndRule   string `json:"endRule,omitempty"`
}

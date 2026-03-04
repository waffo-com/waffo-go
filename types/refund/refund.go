// Package refund provides refund-related types.
package refund

import "github.com/waffo-com/waffo-go/types"

// InquiryRefundParams represents the parameters for querying a refund.
type InquiryRefundParams struct {
	MerchantInfo    types.MerchantInfo `json:"merchantInfo"`
	RefundRequestID string             `json:"refundRequestId"`
}

// InquiryRefundData represents the response data for refund inquiry.
type InquiryRefundData struct {
	RefundRequestID        string `json:"refundRequestId,omitempty"`
	AcquiringRefundOrderID string `json:"acquiringRefundOrderId,omitempty"`
	AcquiringOrderID       string `json:"acquiringOrderId,omitempty"`
	OrigPaymentRequestID   string `json:"origPaymentRequestId,omitempty"`
	RefundStatus           string `json:"refundStatus,omitempty"`
	RefundAmount           string `json:"refundAmount,omitempty"`
	RefundCurrency         string `json:"refundCurrency,omitempty"`
	RemainingRefundAmount  string `json:"remainingRefundAmount,omitempty"`
	UserCurrency           string `json:"userCurrency,omitempty"`
	FinalDealAmount        string `json:"finalDealAmount,omitempty"`
	RefundReason           string `json:"refundReason,omitempty"`
	RefundFailedReason     string `json:"refundFailedReason,omitempty"`
	RefundRequestedAt      string `json:"refundRequestedAt,omitempty"`
	RefundedAt             string `json:"refundedAt,omitempty"`
	RefundUpdatedAt        string `json:"refundUpdatedAt,omitempty"`
	RefundCompletedAt      string `json:"refundCompletedAt,omitempty"`
	RefundSource           string `json:"refundSource,omitempty"`
	ExtendInfo             string `json:"extendInfo,omitempty"`
}

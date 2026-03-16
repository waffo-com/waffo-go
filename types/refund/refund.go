// Package refund provides refund-related types.
package refund

import "github.com/waffo-com/waffo-go/types"

// InquiryRefundParams represents the parameters for querying a refund.
type InquiryRefundParams struct {
	RefundRequestID        string            `json:"refundRequestId"`
	AcquiringRefundOrderID string            `json:"acquiringRefundOrderId,omitempty"`
	ExtraParams            types.ExtraParams `json:"extraParams,omitempty"`
}

// RefundUserInfo represents user information in refund inquiry response.
type RefundUserInfo struct {
	UserType         string `json:"userType,omitempty"`
	UserFirstName    string `json:"userFirstName,omitempty"`
	UserMiddleName   string `json:"userMiddleName,omitempty"`
	UserLastName     string `json:"userLastName,omitempty"`
	Nationality      string `json:"nationality,omitempty"`
	UserEmail        string `json:"userEmail,omitempty"`
	UserPhone        string `json:"userPhone,omitempty"`
	UserBirthDay     string `json:"userBirthDay,omitempty"`
	UserIDType       string `json:"userIDType,omitempty"`
	UserIDNumber     string `json:"userIDNumber,omitempty"`
	UserIDIssueDate  string `json:"userIDIssueDate,omitempty"`
	UserIDExpiryDate string `json:"userIDExpiryDate,omitempty"`
}

// InquiryRefundData represents the response data for refund inquiry.
type InquiryRefundData struct {
	RefundRequestID        string          `json:"refundRequestId,omitempty"`
	MerchantRefundOrderID  string          `json:"merchantRefundOrderId,omitempty"`
	AcquiringOrderID       string          `json:"acquiringOrderId,omitempty"`
	AcquiringRefundOrderID string          `json:"acquiringRefundOrderId,omitempty"`
	OrigPaymentRequestID   string          `json:"origPaymentRequestId,omitempty"`
	RefundAmount           string          `json:"refundAmount,omitempty"`
	RefundStatus           string          `json:"refundStatus,omitempty"`
	RefundReason           string          `json:"refundReason,omitempty"`
	RefundRequestedAt      string          `json:"refundRequestedAt,omitempty"`
	RefundUpdatedAt        string          `json:"refundUpdatedAt,omitempty"`
	RefundFailedReason     string          `json:"refundFailedReason,omitempty"`
	ExtendInfo             string          `json:"extendInfo,omitempty"`
	UserCurrency           string          `json:"userCurrency,omitempty"`
	FinalDealAmount        string          `json:"finalDealAmount,omitempty"`
	RemainingRefundAmount  string          `json:"remainingRefundAmount,omitempty"`
	UserInfo               *RefundUserInfo `json:"userInfo,omitempty"`
	RefundCompletedAt      string          `json:"refundCompletedAt,omitempty"`
	RefundSource           string          `json:"refundSource,omitempty"`
}

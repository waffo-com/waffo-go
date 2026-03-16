// Package order provides order-related types.
package order

import (
	"encoding/json"

	"github.com/waffo-com/waffo-go/types"
	"github.com/waffo-com/waffo-go/types/subscription"
)

// CreateOrderParams represents the parameters for creating an order.
type CreateOrderParams struct {
	PaymentRequestID   string            `json:"paymentRequestId"`
	MerchantOrderID    string            `json:"merchantOrderId"`
	OrderCurrency      string            `json:"orderCurrency"`
	OrderAmount        string            `json:"orderAmount"`
	UserCurrency       string            `json:"userCurrency,omitempty"`
	OrderDescription   string            `json:"orderDescription"`
	OrderRequestedAt   string            `json:"orderRequestedAt,omitempty"`
	OrderExpiredAt     string            `json:"orderExpiredAt,omitempty"`
	SuccessRedirectURL string            `json:"successRedirectUrl,omitempty"`
	FailedRedirectURL  string            `json:"failedRedirectUrl,omitempty"`
	CancelRedirectURL  string            `json:"cancelRedirectUrl,omitempty"`
	NotifyURL          string            `json:"notifyUrl"`
	ExtendInfo         string            `json:"extendInfo,omitempty"`
	MerchantInfo       *MerchantInfo     `json:"merchantInfo,omitempty"`
	UserInfo           *UserInfo         `json:"userInfo"`
	GoodsInfo          *GoodsInfo        `json:"goodsInfo,omitempty"`
	PaymentInfo        *PaymentInfo      `json:"paymentInfo"`
	CardInfo           *CardInfo         `json:"cardInfo,omitempty"`
	PaymentTokenData   *PaymentTokenData `json:"paymentTokenData,omitempty"`
	RiskData           *RiskData         `json:"riskData,omitempty"`
	AddressInfo        *AddressInfo      `json:"addressInfo,omitempty"`
	ExtraParams        types.ExtraParams `json:"extraParams,omitempty"`
}

// CardInfo represents card information for direct card payments.
type CardInfo struct {
	CardNumber      string `json:"cardNumber,omitempty"`
	CardExpiryYear  int    `json:"cardExpiryYear,omitempty"`
	CardExpiryMonth int    `json:"cardExpiryMonth,omitempty"`
	CardCvv         string `json:"cardCvv,omitempty"`
	CardHolderName  string `json:"cardHolderName,omitempty"`
	ThreeDsDecision string `json:"threeDsDecision,omitempty"`
}

// PaymentTokenData represents tokenized payment data.
type PaymentTokenData struct {
	TokenID string `json:"tokenId,omitempty"`
}

// RiskData represents risk control auxiliary data.
type RiskData struct {
	UserType            string `json:"userType,omitempty"`
	UserCategory        string `json:"userCategory,omitempty"`
	UserLegalName       string `json:"userLegalName,omitempty"`
	UserDisplayName     string `json:"userDisplayName,omitempty"`
	UserRegistrationIP  string `json:"userRegistrationIp,omitempty"`
	UserLastSeenIP      string `json:"userLastSeenIp,omitempty"`
	UserIsNew           string `json:"userIsNew,omitempty"`
	UserIsFirstPurchase string `json:"userIsFirstPurchase,omitempty"`
}

// MerchantInfo represents merchant information.
type MerchantInfo struct {
	MerchantID    string `json:"merchantId,omitempty"`
	SubMerchantID string `json:"subMerchantId,omitempty"`
}

// UserInfo represents user information.
type UserInfo struct {
	UserID          string `json:"userId,omitempty"`
	UserEmail       string `json:"userEmail,omitempty"`
	UserPhone       string `json:"userPhone,omitempty"`
	UserCountryCode string `json:"userCountryCode,omitempty"`
	UserTerminal    string `json:"userTerminal,omitempty"`
	UserFirstName   string `json:"userFirstName,omitempty"`
	UserLastName    string `json:"userLastName,omitempty"`
	UserCreatedAt   string `json:"userCreatedAt,omitempty"`
	UserBrowserIP   string `json:"userBrowserIp,omitempty"`
	UserAgent       string `json:"userAgent,omitempty"`
	UserReceiptURL  string `json:"userReceiptUrl,omitempty"`
}

// PaymentInfo represents payment information.
type PaymentInfo struct {
	ProductName              string `json:"productName,omitempty"`
	PayMethodType            string `json:"payMethodType,omitempty"`
	PayMethodName            string `json:"payMethodName,omitempty"`
	PayMethodCountry         string `json:"payMethodCountry,omitempty"`
	PayMethodUserAccountType string `json:"payMethodUserAccountType,omitempty"`
	PayMethodUserAccountNo   string `json:"payMethodUserAccountNo,omitempty"`
	CashierLanguage          string `json:"cashierLanguage,omitempty"`
	UserPaymentAccessToken   string `json:"userPaymentAccessToken,omitempty"`
	CaptureMode              string `json:"captureMode,omitempty"`
	MerchantInitiatedMode    string `json:"merchantInitiatedMode,omitempty"`
}

// GoodsInfo represents goods information.
type GoodsInfo struct {
	GoodsID          string `json:"goodsId,omitempty"`
	GoodsName        string `json:"goodsName,omitempty"`
	GoodsCategory    string `json:"goodsCategory,omitempty"`
	GoodsURL         string `json:"goodsUrl,omitempty"`
	AppName          string `json:"appName,omitempty"`
	SkuName          string `json:"skuName,omitempty"`
	GoodsUniquePrice string `json:"goodsUniquePrice,omitempty"`
	GoodsQuantity    int    `json:"goodsQuantity,omitempty"`
}

// AddressInfo represents address information.
type AddressInfo struct {
	ShippingAddress *Address `json:"shippingAddress,omitempty"`
	BillingAddress  *Address `json:"billingAddress,omitempty"`
}

// Address represents a physical address.
type Address struct {
	Country    string `json:"country,omitempty"`
	State      string `json:"state,omitempty"`
	City       string `json:"city,omitempty"`
	Address1   string `json:"address1,omitempty"`
	Address2   string `json:"address2,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
}

// CreateOrderData represents the response data for order creation.
type CreateOrderData struct {
	PaymentRequestID string `json:"paymentRequestId,omitempty"`
	MerchantOrderID  string `json:"merchantOrderId,omitempty"`
	AcquiringOrderID string `json:"acquiringOrderId,omitempty"`
	OrderStatus      string `json:"orderStatus,omitempty"`
	OrderAction      string `json:"orderAction,omitempty"`
}

// OrderAction represents the action required for order processing.
type OrderAction struct {
	ActionType  string           `json:"actionType,omitempty"`
	WebURL      string           `json:"webUrl,omitempty"`
	DeeplinkURL string           `json:"deeplinkUrl,omitempty"`
	ActionData  *OrderActionData `json:"actionData,omitempty"`
}

// OrderActionData represents additional data for order action.
type OrderActionData struct {
	// Add fields as needed
}

// FetchRedirectURL returns the redirect URL from the order action.
// It parses the OrderAction JSON string and returns webUrl or deeplinkUrl.
func (d *CreateOrderData) FetchRedirectURL() string {
	if d.OrderAction == "" {
		return ""
	}

	// Parse the JSON string
	var action OrderAction
	if err := json.Unmarshal([]byte(d.OrderAction), &action); err != nil {
		return ""
	}

	// Return deeplink URL if action type is DEEPLINK and it's available
	if action.ActionType == "DEEPLINK" && action.DeeplinkURL != "" {
		return action.DeeplinkURL
	}

	// Otherwise return web URL
	return action.WebURL
}

// InquiryOrderParams represents the parameters for querying an order.
// Provide paymentRequestId or acquiringOrderId.
type InquiryOrderParams struct {
	PaymentRequestID string            `json:"paymentRequestId,omitempty"`
	AcquiringOrderID string            `json:"acquiringOrderId,omitempty"`
	ExtraParams      types.ExtraParams `json:"extraParams,omitempty"`
}

// InquiryOrderData represents the response data for order inquiry.
type InquiryOrderData struct {
	PaymentRequestID  string                         `json:"paymentRequestId,omitempty"`
	MerchantOrderID   string                         `json:"merchantOrderId,omitempty"`
	AcquiringOrderID  string                         `json:"acquiringOrderId,omitempty"`
	OrderStatus       string                         `json:"orderStatus,omitempty"`
	OrderAction       string                         `json:"orderAction,omitempty"`
	OrderCurrency     string                         `json:"orderCurrency,omitempty"`
	OrderAmount       string                         `json:"orderAmount,omitempty"`
	FinalDealAmount   string                         `json:"finalDealAmount,omitempty"`
	OrderDescription  string                         `json:"orderDescription,omitempty"`
	MerchantInfo      *MerchantInfo                  `json:"merchantInfo,omitempty"`
	UserInfo          *UserInfo                      `json:"userInfo,omitempty"`
	GoodsInfo         *GoodsInfo                     `json:"goodsInfo,omitempty"`
	PaymentInfo       *PaymentInfo                   `json:"paymentInfo,omitempty"`
	AddressInfo       *AddressInfo                   `json:"addressInfo,omitempty"`
	OrderRequestedAt  string                         `json:"orderRequestedAt,omitempty"`
	OrderExpiredAt    string                         `json:"orderExpiredAt,omitempty"`
	OrderUpdatedAt    string                         `json:"orderUpdatedAt,omitempty"`
	OrderCompletedAt  string                         `json:"orderCompletedAt,omitempty"`
	RefundExpiryAt    string                         `json:"refundExpiryAt,omitempty"`
	CancelRedirectURL string                         `json:"cancelRedirectUrl,omitempty"`
	OrderFailedReason string                         `json:"orderFailedReason,omitempty"`
	ExtendInfo        string                         `json:"extendInfo,omitempty"`
	UserCurrency      string                         `json:"userCurrency,omitempty"`
	SubscriptionInfo  *subscription.SubscriptionInfo `json:"subscriptionInfo,omitempty"`
}

// CancelOrderParams represents the parameters for canceling an order.
type CancelOrderParams struct {
	PaymentRequestID string            `json:"paymentRequestId,omitempty"`
	AcquiringOrderID string            `json:"acquiringOrderId,omitempty"`
	MerchantID       string            `json:"merchantId,omitempty"`
	OrderRequestedAt string            `json:"orderRequestedAt,omitempty"`
	ExtraParams      types.ExtraParams `json:"extraParams,omitempty"`
}

// CancelOrderData represents the response data for order cancellation.
type CancelOrderData struct {
	PaymentRequestID string `json:"paymentRequestId,omitempty"`
	MerchantOrderID  string `json:"merchantOrderId,omitempty"`
	AcquiringOrderID string `json:"acquiringOrderId,omitempty"`
	OrderStatus      string `json:"orderStatus,omitempty"`
}

// RefundOrderParams represents the parameters for refunding an order.
type RefundOrderParams struct {
	RefundRequestID       string            `json:"refundRequestId"`
	AcquiringOrderID      string            `json:"acquiringOrderId,omitempty"`
	MerchantRefundOrderID string            `json:"merchantRefundOrderId,omitempty"`
	MerchantID            string            `json:"merchantId,omitempty"`
	RequestedAt           string            `json:"requestedAt,omitempty"`
	RefundAmount          string            `json:"refundAmount"`
	RefundReason          string            `json:"refundReason,omitempty"`
	NotifyURL             string            `json:"refundNotifyUrl,omitempty"`
	ExtendInfo            string            `json:"extendInfo,omitempty"`
	RefundSource          string            `json:"refundSource,omitempty"`
	UserInfo              *RefundUserInfo   `json:"userInfo,omitempty"`
	ExtraParams           types.ExtraParams `json:"extraParams,omitempty"`
}

// RefundOrderData represents the response data for order refund.
type RefundOrderData struct {
	RefundRequestID        string `json:"refundRequestId,omitempty"`
	MerchantRefundOrderID  string `json:"merchantRefundOrderId,omitempty"`
	AcquiringOrderID       string `json:"acquiringOrderId,omitempty"`
	AcquiringRefundOrderID string `json:"acquiringRefundOrderId,omitempty"`
	RefundAmount           string `json:"refundAmount,omitempty"`
	RefundStatus           string `json:"refundStatus,omitempty"`
	RemainingRefundAmount  string `json:"remainingRefundAmount,omitempty"`
	RefundSource           string `json:"refundSource,omitempty"`
}

// CaptureOrderParams represents the parameters for capturing a pre-authorized payment.
type CaptureOrderParams struct {
	PaymentRequestID   string            `json:"paymentRequestId,omitempty"`
	AcquiringOrderID   string            `json:"acquiringOrderId,omitempty"`
	MerchantID         string            `json:"merchantId,omitempty"`
	CaptureRequestedAt string            `json:"captureRequestedAt,omitempty"`
	CaptureAmount      string            `json:"captureAmount"`
	ExtraParams        types.ExtraParams `json:"extraParams,omitempty"`
}

// CaptureOrderData represents the response data for order capture.
type CaptureOrderData struct {
	PaymentRequestID string `json:"paymentRequestId,omitempty"`
	MerchantOrderID  string `json:"merchantOrderId,omitempty"`
	AcquiringOrderID string `json:"acquiringOrderId,omitempty"`
	OrderStatus      string `json:"orderStatus,omitempty"`
}

// RefundUserInfo represents user information required for refunds with specific payment methods.
type RefundUserInfo struct {
	UserType        string          `json:"userType,omitempty"`
	UserFirstName   string          `json:"userFirstName,omitempty"`
	UserMiddleName  string          `json:"userMiddleName,omitempty"`
	UserLastName    string          `json:"userLastName,omitempty"`
	Nationality     string          `json:"nationality,omitempty"`
	UserEmail       string          `json:"userEmail,omitempty"`
	UserPhone       string          `json:"userPhone,omitempty"`
	UserBirthDay    string          `json:"userBirthDay,omitempty"`
	UserIDType      string          `json:"userIDType,omitempty"`
	UserIDNumber    string          `json:"userIDNumber,omitempty"`
	UserIDIssueDate string          `json:"userIDIssueDate,omitempty"`
	UserIDExpiryDate string         `json:"userIDExpiryDate,omitempty"`
	UserBankInfo    *RefundUserBankInfo `json:"userBankInfo,omitempty"`
}

// RefundUserBankInfo represents bank account information for refund user.
type RefundUserBankInfo struct {
	BankAccountNo string `json:"bankAccountNo,omitempty"`
	BankCode      string `json:"bankCode,omitempty"`
	BankName      string `json:"bankName,omitempty"`
	BankCity      string `json:"bankCity,omitempty"`
	BankBranch    string `json:"bankBranch,omitempty"`
}

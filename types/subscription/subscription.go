// Package subscription provides subscription-related types.
package subscription

import (
	"encoding/json"
	"strings"

	"github.com/waffo-com/waffo-go/types"
)

// SubscriptionInfo represents subscription information in payment order response and webhook notifications.
type SubscriptionInfo struct {
	SubscriptionRequest string `json:"subscriptionRequest,omitempty"`
	MerchantRequest     string `json:"merchantRequest,omitempty"`
	SubscriptionID      string `json:"subscriptionId,omitempty"`
	Period              string `json:"period,omitempty"`
}

// CreateSubscriptionParams represents the parameters for creating a subscription.
type CreateSubscriptionParams struct {
	SubscriptionRequest       string                    `json:"subscriptionRequest"`
	MerchantSubscriptionID    string                    `json:"merchantSubscriptionId,omitempty"`
	Currency                  string                    `json:"currency"`
	Amount                    string                    `json:"amount"`
	UserCurrency              string                    `json:"userCurrency,omitempty"`
	ProductInfo               *ProductInfo              `json:"productInfo,omitempty"`
	MerchantInfo              *SubscriptionMerchantInfo `json:"merchantInfo,omitempty"`
	UserInfo                  *SubscriptionUserInfo     `json:"userInfo"`
	GoodsInfo                 *SubscriptionGoodsInfo    `json:"goodsInfo,omitempty"`
	AddressInfo               *SubscriptionAddressInfo  `json:"addressInfo,omitempty"`
	PaymentInfo               *SubscriptionPaymentInfo  `json:"paymentInfo"`
	RiskData                  *SubscriptionRiskData     `json:"riskData,omitempty"`
	RequestedAt               string                    `json:"requestedAt,omitempty"`
	OrderExpiredAt            string                    `json:"orderExpiredAt,omitempty"`
	SuccessRedirectURL        string                    `json:"successRedirectUrl,omitempty"`
	FailedRedirectURL         string                    `json:"failedRedirectUrl,omitempty"`
	CancelRedirectURL         string                    `json:"cancelRedirectUrl,omitempty"`
	SubscriptionManagementURL string                    `json:"subscriptionManagementUrl,omitempty"`
	NotifyURL                 string                    `json:"notifyUrl,omitempty"`
	ExtendInfo                string                    `json:"extendInfo,omitempty"`
	Metadata                  string                    `json:"metadata,omitempty"`
	ExtraParams               types.ExtraParams         `json:"extraParams,omitempty"`
}

// ProductInfo represents product/billing configuration.
type ProductInfo struct {
	Description         string `json:"description,omitempty"`         // REQUIRED, max 128
	PeriodType          string `json:"periodType,omitempty"`          // REQUIRED, DAILY/WEEKLY/MONTHLY
	PeriodInterval      string `json:"periodInterval,omitempty"`      // REQUIRED, max 12
	NumberOfPeriod      string `json:"numberOfPeriod,omitempty"`      // max 24
	TrialPeriodAmount   string `json:"trialPeriodAmount,omitempty"`   // max 24
	NumberOfTrialPeriod string `json:"numberOfTrialPeriod,omitempty"` // max 12
	TrialPeriodType     string `json:"trialPeriodType,omitempty"`     // DAILY/WEEKLY/MONTHLY
	TrialPeriodInterval string `json:"trialPeriodInterval,omitempty"` // max 12
	StartDateTime       string `json:"startDateTime,omitempty"`       // Response only, ISO 8601
	EndDateTime         string `json:"endDateTime,omitempty"`         // Response only, ISO 8601
	NextPaymentDateTime string `json:"nextPaymentDateTime,omitempty"` // Response only, ISO 8601
	CurrentPeriod       string `json:"currentPeriod,omitempty"`       // Response only
}

// SubscriptionMerchantInfo represents merchant information for subscriptions.
type SubscriptionMerchantInfo struct {
	MerchantID    string `json:"merchantId,omitempty"`
	SubMerchantID string `json:"subMerchantId,omitempty"`
}

// SubscriptionUserInfo represents user information for subscriptions.
type SubscriptionUserInfo struct {
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
}

// SubscriptionGoodsInfo represents goods information for subscriptions.
type SubscriptionGoodsInfo struct {
	GoodsID          string `json:"goodsId,omitempty"`
	GoodsName        string `json:"goodsName,omitempty"`
	GoodsCategory    string `json:"goodsCategory,omitempty"`
	GoodsURL         string `json:"goodsUrl,omitempty"`
	AppName          string `json:"appName,omitempty"`
	SkuName          string `json:"skuName,omitempty"`
	GoodsUniquePrice string `json:"goodsUniquePrice,omitempty"`
	GoodsQuantity    int    `json:"goodsQuantity,omitempty"`
}

// SubscriptionAddressInfo represents address information for subscriptions.
type SubscriptionAddressInfo struct {
	ShippingAddress *SubscriptionAddress `json:"shippingAddress,omitempty"`
	BillingAddress  *SubscriptionAddress `json:"billingAddress,omitempty"`
}

// SubscriptionAddress represents a physical address.
type SubscriptionAddress struct {
	Country    string `json:"country,omitempty"`
	State      string `json:"state,omitempty"`
	City       string `json:"city,omitempty"`
	Address1   string `json:"address1,omitempty"`
	Address2   string `json:"address2,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
}

// SubscriptionPaymentInfo represents payment information for subscriptions.
type SubscriptionPaymentInfo struct {
	ProductName              string `json:"productName,omitempty"`
	PayMethodType            string `json:"payMethodType,omitempty"`
	PayMethodName            string `json:"payMethodName,omitempty"`
	PayMethodProperties      string `json:"payMethodProperties,omitempty"`
	PayMethodPublicUid       string `json:"payMethodPublicUid,omitempty"`
	PayMethodUserAccessToken string `json:"payMethodUserAccessToken,omitempty"`
	PayMethodUserAccountType string `json:"payMethodUserAccountType,omitempty"`
	PayMethodUserAccountNo   string `json:"payMethodUserAccountNo,omitempty"`
	PayMethodResponse        string `json:"payMethodResponse,omitempty"`
	CashierLanguage          string `json:"cashierLanguage,omitempty"`
}

// SubscriptionRiskData represents risk control data for subscriptions.
type SubscriptionRiskData struct {
	DeviceType    string `json:"deviceType,omitempty"`
	DeviceId      string `json:"deviceId,omitempty"`
	DeviceTokenId string `json:"deviceTokenId,omitempty"`
}

// CreateSubscriptionData represents the response data for subscription creation.
type CreateSubscriptionData struct {
	SubscriptionRequest      string `json:"subscriptionRequest,omitempty"`
	MerchantSubscriptionID   string `json:"merchantSubscriptionId,omitempty"`
	SubscriptionID           string `json:"subscriptionId,omitempty"`
	PayMethodSubscriptionID  string `json:"payMethodSubscriptionId,omitempty"`
	SubscriptionStatus       string `json:"subscriptionStatus,omitempty"`
	SubscriptionAction       string `json:"subscriptionAction,omitempty"`
}

// FetchRedirectURL returns the redirect URL from the subscription action.
func (d *CreateSubscriptionData) FetchRedirectURL() string {
	if d.SubscriptionAction == "" {
		return ""
	}

	trimmed := strings.TrimSpace(d.SubscriptionAction)

	// If it looks like a URL, return directly
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}

	// Try to parse as JSON and extract webUrl
	var action struct {
		WebURL      string `json:"webUrl"`
		DeeplinkURL string `json:"deeplinkUrl"`
	}
	if err := json.Unmarshal([]byte(trimmed), &action); err == nil {
		if action.WebURL != "" {
			return action.WebURL
		}
		if action.DeeplinkURL != "" {
			return action.DeeplinkURL
		}
	}

	return ""
}

// InquirySubscriptionParams represents the parameters for querying a subscription.
type InquirySubscriptionParams struct {
	MerchantInfo           *SubscriptionMerchantInfo `json:"merchantInfo,omitempty"`
	SubscriptionRequest    string                    `json:"subscriptionRequest,omitempty"`
	MerchantSubscriptionID string                    `json:"merchantSubscriptionId,omitempty"`
	SubscriptionID         string                    `json:"subscriptionId,omitempty"`
	ExtraParams            types.ExtraParams         `json:"extraParams,omitempty"`
}

// InquirySubscriptionData represents the response data for subscription inquiry.
type InquirySubscriptionData struct {
	SubscriptionRequest       string                    `json:"subscriptionRequest,omitempty"`
	SubscriptionID            string                    `json:"subscriptionId,omitempty"`
	MerchantSubscriptionID    string                    `json:"merchantSubscriptionId,omitempty"`
	PayMethodSubscriptionID   string                    `json:"payMethodSubscriptionId,omitempty"`
	SubscriptionStatus        string                    `json:"subscriptionStatus,omitempty"`
	SubscriptionAction        string                    `json:"subscriptionAction,omitempty"`
	Amount                    string                    `json:"amount,omitempty"`
	Currency                  string                    `json:"currency,omitempty"`
	UserCurrency              string                    `json:"userCurrency,omitempty"`
	ProductInfo               *ProductInfo              `json:"productInfo,omitempty"`
	MerchantInfo              *SubscriptionMerchantInfo `json:"merchantInfo,omitempty"`
	UserInfo                  *SubscriptionUserInfo     `json:"userInfo,omitempty"`
}

// CancelSubscriptionParams represents the parameters for canceling a subscription.
type CancelSubscriptionParams struct {
	SubscriptionID string            `json:"subscriptionId,omitempty"`
	MerchantID     string            `json:"merchantId,omitempty"`
	RequestedAt    string            `json:"requestedAt,omitempty"`
	ExtraParams    types.ExtraParams `json:"extraParams,omitempty"`
}

// CancelSubscriptionData represents the response data for subscription cancellation.
type CancelSubscriptionData struct {
	SubscriptionID     string `json:"subscriptionId,omitempty"`
	SubscriptionStatus string `json:"subscriptionStatus,omitempty"`
}

// ManageSubscriptionParams represents the parameters for managing a subscription.
type ManageSubscriptionParams struct {
	MerchantInfo           *SubscriptionMerchantInfo `json:"merchantInfo,omitempty"`
	SubscriptionRequest    string                    `json:"subscriptionRequest,omitempty"`
	MerchantSubscriptionID string                    `json:"merchantSubscriptionId,omitempty"`
	SubscriptionID         string                    `json:"subscriptionId,omitempty"`
	ReturnURL              string                    `json:"returnUrl,omitempty"`
	ExtraParams            types.ExtraParams         `json:"extraParams,omitempty"`
}

// ManageSubscriptionData represents the response data for subscription management.
type ManageSubscriptionData struct {
	ManageURL string `json:"manageUrl,omitempty"`
}

// SubscriptionChangeProductInfo represents product info for subscription change.
type SubscriptionChangeProductInfo struct {
	Description         string `json:"description"`
	PeriodType          string `json:"periodType"`
	PeriodInterval      string `json:"periodInterval"`
	Amount              string `json:"amount"`
	NumberOfPeriod      string `json:"numberOfPeriod,omitempty"`
	TrialPeriodAmount   string `json:"trialPeriodAmount,omitempty"`
	NumberOfTrialPeriod string `json:"numberOfTrialPeriod,omitempty"`
	TrialPeriodType     string `json:"trialPeriodType,omitempty"`
	TrialPeriodInterval string `json:"trialPeriodInterval,omitempty"`
}

// ChangeSubscriptionParams represents the parameters for changing a subscription.
type ChangeSubscriptionParams struct {
	SubscriptionRequest       string                          `json:"subscriptionRequest"`
	MerchantSubscriptionID    string                          `json:"merchantSubscriptionId,omitempty"`
	OriginSubscriptionRequest string                          `json:"originSubscriptionRequest"`
	RemainingAmount           string                          `json:"remainingAmount,omitempty"`
	Currency                  string                          `json:"currency"`
	UserCurrency              string                          `json:"userCurrency,omitempty"`
	RequestedAt               string                          `json:"requestedAt"`
	SuccessRedirectURL        string                          `json:"successRedirectUrl,omitempty"`
	FailedRedirectURL         string                          `json:"failedRedirectUrl,omitempty"`
	CancelRedirectURL         string                          `json:"cancelRedirectUrl,omitempty"`
	NotifyURL                 string                          `json:"notifyUrl"`
	SubscriptionManagementURL string                          `json:"subscriptionManagementUrl,omitempty"`
	ExtendInfo                string                          `json:"extendInfo,omitempty"`
	OrderExpiredAt            string                          `json:"orderExpiredAt,omitempty"`
	ProductInfoList           []SubscriptionChangeProductInfo `json:"productInfoList"`
	MerchantInfo              *SubscriptionMerchantInfo       `json:"merchantInfo"`
	UserInfo                  *SubscriptionUserInfo           `json:"userInfo"`
	GoodsInfo                 *SubscriptionGoodsInfo          `json:"goodsInfo"`
	AddressInfo               *SubscriptionAddressInfo        `json:"addressInfo,omitempty"`
	PaymentInfo               *SubscriptionPaymentInfo        `json:"paymentInfo"`
	RiskData                  *SubscriptionRiskData           `json:"riskData,omitempty"`
	ExtraParams               types.ExtraParams               `json:"extraParams,omitempty"`
}

// ChangeSubscriptionData represents the response data for subscription change.
type ChangeSubscriptionData struct {
	SubscriptionRequest  string `json:"subscriptionRequest,omitempty"`
	SubscriptionID       string `json:"subscriptionId,omitempty"`
	SubscriptionChangeStatus string `json:"subscriptionChangeStatus,omitempty"`
	SubscriptionAction   string `json:"subscriptionAction,omitempty"`
}

// FetchRedirectURL returns the redirect URL from the subscription action.
func (d *ChangeSubscriptionData) FetchRedirectURL() string {
	if d.SubscriptionAction == "" {
		return ""
	}

	trimmed := strings.TrimSpace(d.SubscriptionAction)

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}

	var action struct {
		WebURL string `json:"webUrl"`
	}
	if err := json.Unmarshal([]byte(trimmed), &action); err == nil && action.WebURL != "" {
		return action.WebURL
	}

	return ""
}

// ChangeInquiryParams represents the parameters for querying a subscription change.
type ChangeInquiryParams struct {
	MerchantInfo        *SubscriptionMerchantInfo `json:"merchantInfo,omitempty"`
	SubscriptionRequest string                    `json:"subscriptionRequest"`
	ExtraParams         types.ExtraParams         `json:"extraParams,omitempty"`
}

// ChangeInquiryData represents the response data for subscription change inquiry.
type ChangeInquiryData struct {
	SubscriptionRequest       string       `json:"subscriptionRequest,omitempty"`
	OriginSubscriptionRequest string       `json:"originSubscriptionRequest,omitempty"`
	SubscriptionID            string       `json:"subscriptionId,omitempty"`
	SubscriptionChangeStatus  string       `json:"subscriptionChangeStatus,omitempty"`
	ProductInfo               *ProductInfo `json:"productInfo,omitempty"`
}

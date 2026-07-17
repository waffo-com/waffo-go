// Package wallet provides x402 stablecoin wallet types.
package wallet

import "github.com/waffo-com/waffo-go/types"

// WalletInquiryParams represents the parameters for an x402 stablecoin
// receiving-address inquiry (POST /api/v1/wallet/inquiry).
//
// MerchantInfo is auto-injected by the HTTP client before signing, so it is
// declared optional here; PaymentRequestID and OrderCurrency are required and
// are not injected, so they carry no omitempty.
type WalletInquiryParams struct {
	PaymentRequestID string              `json:"paymentRequestId"`
	MerchantInfo     *types.MerchantInfo `json:"merchantInfo,omitempty"`
	OrderCurrency    string              `json:"orderCurrency"`
	ExtraParams      types.ExtraParams   `json:"extraParams,omitempty"`
}

// WalletInquiryData represents the response data for a wallet inquiry.
type WalletInquiryData struct {
	Network string `json:"network,omitempty"`
	Asset   string `json:"asset,omitempty"`
	PayTo   string `json:"payTo,omitempty"`
}

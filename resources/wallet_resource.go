package resources

import (
	"context"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/wallet"
)

const walletBasePath = "/wallet"

// WalletResource provides methods for x402 stablecoin wallet operations.
type WalletResource struct {
	*BaseResource
}

// NewWalletResource creates a new WalletResource.
func NewWalletResource(httpClient *core.WaffoHttpClient) *WalletResource {
	return &WalletResource{
		BaseResource: NewBaseResource(httpClient, walletBasePath),
	}
}

// Inquiry queries the x402 stablecoin receiving address for an order.
func (r *WalletResource) Inquiry(ctx context.Context, params *wallet.WalletInquiryParams, opts *config.RequestOptions) (*core.ApiResponse[wallet.WalletInquiryData], error) {
	resp, err := core.PostWithResponse[wallet.WalletInquiryData](r.httpClient, ctx, r.GetPath("/inquiry"), params, opts)
	if err != nil {
		return core.Error[wallet.WalletInquiryData]("E0001", err.Error()), nil
	}
	return resp, nil
}

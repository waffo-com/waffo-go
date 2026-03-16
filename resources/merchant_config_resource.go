package resources

import (
	"context"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/merchant"
)

const merchantConfigBasePath = "/merchantconfig"

// MerchantConfigResource provides methods for merchant configuration.
type MerchantConfigResource struct {
	*BaseResource
}

// NewMerchantConfigResource creates a new MerchantConfigResource.
func NewMerchantConfigResource(httpClient *core.WaffoHttpClient) *MerchantConfigResource {
	return &MerchantConfigResource{
		BaseResource: NewBaseResource(httpClient, merchantConfigBasePath),
	}
}

// Inquiry queries the merchant configuration.
func (r *MerchantConfigResource) Inquiry(ctx context.Context, params *merchant.InquiryMerchantConfigParams, opts *config.RequestOptions) (*core.ApiResponse[merchant.InquiryMerchantConfigData], error) {
	resp, err := core.PostWithResponse[merchant.InquiryMerchantConfigData](r.httpClient, ctx, r.GetPath("/inquiry"), params, opts)
	if err != nil {
		return core.Error[merchant.InquiryMerchantConfigData]("E0001", err.Error()), nil
	}
	return resp, nil
}

package resources

import (
	"context"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/merchant"
)

const payMethodConfigBasePath = "/paymethodconfig"

// PayMethodConfigResource provides methods for payment method configuration.
type PayMethodConfigResource struct {
	*BaseResource
}

// NewPayMethodConfigResource creates a new PayMethodConfigResource.
func NewPayMethodConfigResource(httpClient *core.WaffoHttpClient) *PayMethodConfigResource {
	return &PayMethodConfigResource{
		BaseResource: NewBaseResource(httpClient, payMethodConfigBasePath),
	}
}

// Inquiry queries the payment method configuration.
func (r *PayMethodConfigResource) Inquiry(ctx context.Context, params *merchant.InquiryPayMethodConfigParams, opts *config.RequestOptions) (*core.ApiResponse[merchant.InquiryPayMethodConfigData], error) {
	resp, err := core.PostWithResponse[merchant.InquiryPayMethodConfigData](r.httpClient, ctx, r.GetPath("/inquiry"), params, opts)
	if err != nil {
		return core.Error[merchant.InquiryPayMethodConfigData]("E0001", err.Error()), nil
	}
	return resp, nil
}

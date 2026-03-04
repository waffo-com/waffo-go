package resources

import (
	"context"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/refund"
)

const refundBasePath = "/refund"

// RefundResource provides methods for refund management.
type RefundResource struct {
	*BaseResource
}

// NewRefundResource creates a new RefundResource.
func NewRefundResource(httpClient *core.WaffoHttpClient) *RefundResource {
	return &RefundResource{
		BaseResource: NewBaseResource(httpClient, refundBasePath),
	}
}

// Inquiry queries the status of a refund.
func (r *RefundResource) Inquiry(ctx context.Context, params *refund.InquiryRefundParams, opts *config.RequestOptions) (*core.ApiResponse[refund.InquiryRefundData], error) {
	resp, err := core.PostWithResponse[refund.InquiryRefundData](r.httpClient, ctx, r.GetPath("/inquiry"), params, opts)
	if err != nil {
		return core.Error[refund.InquiryRefundData]("E0001", err.Error()), nil
	}
	return resp, nil
}

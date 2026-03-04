package resources

import (
	"context"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/order"
)

const orderBasePath = "/order"

// OrderResource provides methods for order management.
type OrderResource struct {
	*BaseResource
}

// NewOrderResource creates a new OrderResource.
func NewOrderResource(httpClient *core.WaffoHttpClient) *OrderResource {
	return &OrderResource{
		BaseResource: NewBaseResource(httpClient, orderBasePath),
	}
}

// Create creates a new payment order.
func (r *OrderResource) Create(ctx context.Context, params *order.CreateOrderParams, opts *config.RequestOptions) (*core.ApiResponse[order.CreateOrderData], error) {
	return core.PostWithResponse[order.CreateOrderData](r.httpClient, ctx, r.GetPath("/create"), params, opts)
}

// Inquiry queries the status of an order.
func (r *OrderResource) Inquiry(ctx context.Context, params *order.InquiryOrderParams, opts *config.RequestOptions) (*core.ApiResponse[order.InquiryOrderData], error) {
	resp, err := core.PostWithResponse[order.InquiryOrderData](r.httpClient, ctx, r.GetPath("/inquiry"), params, opts)
	if err != nil {
		// For inquiry, return error response instead of throwing
		return core.Error[order.InquiryOrderData]("E0001", err.Error()), nil
	}
	return resp, nil
}

// Cancel cancels an unpaid order.
func (r *OrderResource) Cancel(ctx context.Context, params *order.CancelOrderParams, opts *config.RequestOptions) (*core.ApiResponse[order.CancelOrderData], error) {
	return core.PostWithResponse[order.CancelOrderData](r.httpClient, ctx, r.GetPath("/cancel"), params, opts)
}

// Refund requests a refund for a paid order.
func (r *OrderResource) Refund(ctx context.Context, params *order.RefundOrderParams, opts *config.RequestOptions) (*core.ApiResponse[order.RefundOrderData], error) {
	return core.PostWithResponse[order.RefundOrderData](r.httpClient, ctx, r.GetPath("/refund"), params, opts)
}

// Capture captures a pre-authorized payment.
func (r *OrderResource) Capture(ctx context.Context, params *order.CaptureOrderParams, opts *config.RequestOptions) (*core.ApiResponse[order.CaptureOrderData], error) {
	resp, err := core.PostWithResponse[order.CaptureOrderData](r.httpClient, ctx, r.GetPath("/capture"), params, opts)
	if err != nil {
		return core.Error[order.CaptureOrderData]("E0001", err.Error()), nil
	}
	return resp, nil
}

package resources

import (
	"context"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/subscription"
)

const subscriptionBasePath = "/subscription"

// SubscriptionResource provides methods for subscription management.
type SubscriptionResource struct {
	*BaseResource
}

// NewSubscriptionResource creates a new SubscriptionResource.
func NewSubscriptionResource(httpClient *core.WaffoHttpClient) *SubscriptionResource {
	return &SubscriptionResource{
		BaseResource: NewBaseResource(httpClient, subscriptionBasePath),
	}
}

// Create creates a new subscription.
func (r *SubscriptionResource) Create(ctx context.Context, params *subscription.CreateSubscriptionParams, opts *config.RequestOptions) (*core.ApiResponse[subscription.CreateSubscriptionData], error) {
	return core.PostWithResponse[subscription.CreateSubscriptionData](r.httpClient, ctx, r.GetPath("/create"), params, opts)
}

// Inquiry queries the status of a subscription.
func (r *SubscriptionResource) Inquiry(ctx context.Context, params *subscription.InquirySubscriptionParams, opts *config.RequestOptions) (*core.ApiResponse[subscription.InquirySubscriptionData], error) {
	resp, err := core.PostWithResponse[subscription.InquirySubscriptionData](r.httpClient, ctx, r.GetPath("/inquiry"), params, opts)
	if err != nil {
		return core.Error[subscription.InquirySubscriptionData]("E0001", err.Error()), nil
	}
	return resp, nil
}

// Cancel cancels a subscription.
func (r *SubscriptionResource) Cancel(ctx context.Context, params *subscription.CancelSubscriptionParams, opts *config.RequestOptions) (*core.ApiResponse[subscription.CancelSubscriptionData], error) {
	return core.PostWithResponse[subscription.CancelSubscriptionData](r.httpClient, ctx, r.GetPath("/cancel"), params, opts)
}

// Manage gets the subscription management URL.
func (r *SubscriptionResource) Manage(ctx context.Context, params *subscription.ManageSubscriptionParams, opts *config.RequestOptions) (*core.ApiResponse[subscription.ManageSubscriptionData], error) {
	resp, err := core.PostWithResponse[subscription.ManageSubscriptionData](r.httpClient, ctx, r.GetPath("/manage"), params, opts)
	if err != nil {
		return core.Error[subscription.ManageSubscriptionData]("E0001", err.Error()), nil
	}
	return resp, nil
}

// Change changes (upgrades/downgrades) a subscription.
func (r *SubscriptionResource) Change(ctx context.Context, params *subscription.ChangeSubscriptionParams, opts *config.RequestOptions) (*core.ApiResponse[subscription.ChangeSubscriptionData], error) {
	return core.PostWithResponse[subscription.ChangeSubscriptionData](r.httpClient, ctx, r.GetPath("/change"), params, opts)
}

// ChangeInquiry queries the status of a subscription change.
func (r *SubscriptionResource) ChangeInquiry(ctx context.Context, params *subscription.ChangeInquiryParams, opts *config.RequestOptions) (*core.ApiResponse[subscription.ChangeInquiryData], error) {
	resp, err := core.PostWithResponse[subscription.ChangeInquiryData](r.httpClient, ctx, r.GetPath("/change/inquiry"), params, opts)
	if err != nil {
		return core.Error[subscription.ChangeInquiryData]("E0001", err.Error()), nil
	}
	return resp, nil
}

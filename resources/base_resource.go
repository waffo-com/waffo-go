// Package resources provides API resource implementations.
package resources

import (
	"github.com/waffo-com/waffo-go/core"
)

// BaseResource is the base class for all API resources.
type BaseResource struct {
	httpClient *core.WaffoHttpClient
	basePath   string
}

// NewBaseResource creates a new BaseResource.
func NewBaseResource(httpClient *core.WaffoHttpClient, basePath string) *BaseResource {
	return &BaseResource{
		httpClient: httpClient,
		basePath:   basePath,
	}
}

// GetPath returns the full path for an endpoint.
func (r *BaseResource) GetPath(endpoint string) string {
	return r.basePath + endpoint
}

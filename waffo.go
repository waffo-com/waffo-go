// Package waffo provides the official Go SDK for Waffo Payment Services.
//
// Quick Start:
//
//	cfg, err := config.NewConfigBuilder().
//	    APIKey("your-api-key").
//	    PrivateKey("your-private-key-base64").
//	    WaffoPublicKey("waffo-public-key-base64").
//	    Environment(config.Sandbox).
//	    Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	waffo := waffo.New(cfg)
//
//	// Create an order
//	resp, err := waffo.Order().Create(ctx, &order.CreateOrderParams{
//	    MerchantOrderID: "ORDER-123",
//	    Amount:          "100.00",
//	    Currency:        "USD",
//	}, nil)
package waffo

import (
	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/resources"
	"github.com/waffo-com/waffo-go/utils"
)

// Waffo is the main entry point for the Waffo SDK.
type Waffo struct {
	config     *config.WaffoConfig
	httpClient *core.WaffoHttpClient

	// Lazy-initialized resources
	order            *resources.OrderResource
	subscription     *resources.SubscriptionResource
	refund           *resources.RefundResource
	merchantConfig   *resources.MerchantConfigResource
	payMethodConfig  *resources.PayMethodConfigResource
}

// New creates a new Waffo SDK instance.
func New(cfg *config.WaffoConfig) *Waffo {
	return &Waffo{
		config:     cfg,
		httpClient: core.NewWaffoHttpClient(cfg),
	}
}

// FromEnv creates a new Waffo SDK instance from environment variables.
//
// Environment variables:
//   - WAFFO_API_KEY: API key (required)
//   - WAFFO_PRIVATE_KEY: Private key in Base64 (required)
//   - WAFFO_PUBLIC_KEY: Waffo's public key in Base64 (required)
//   - WAFFO_ENVIRONMENT: "SANDBOX" or "PRODUCTION" (default: SANDBOX)
//   - WAFFO_MERCHANT_ID: Merchant ID for auto-injection (optional)
func FromEnv() (*Waffo, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, err
	}
	return New(cfg), nil
}

// Order returns the OrderResource for order management.
func (w *Waffo) Order() *resources.OrderResource {
	if w.order == nil {
		w.order = resources.NewOrderResource(w.httpClient)
	}
	return w.order
}

// Subscription returns the SubscriptionResource for subscription management.
func (w *Waffo) Subscription() *resources.SubscriptionResource {
	if w.subscription == nil {
		w.subscription = resources.NewSubscriptionResource(w.httpClient)
	}
	return w.subscription
}

// Refund returns the RefundResource for refund queries.
func (w *Waffo) Refund() *resources.RefundResource {
	if w.refund == nil {
		w.refund = resources.NewRefundResource(w.httpClient)
	}
	return w.refund
}

// MerchantConfig returns the MerchantConfigResource for merchant configuration.
func (w *Waffo) MerchantConfig() *resources.MerchantConfigResource {
	if w.merchantConfig == nil {
		w.merchantConfig = resources.NewMerchantConfigResource(w.httpClient)
	}
	return w.merchantConfig
}

// PayMethodConfig returns the PayMethodConfigResource for payment method configuration.
func (w *Waffo) PayMethodConfig() *resources.PayMethodConfigResource {
	if w.payMethodConfig == nil {
		w.payMethodConfig = resources.NewPayMethodConfigResource(w.httpClient)
	}
	return w.payMethodConfig
}

// Webhook returns a new WebhookHandler for processing webhook notifications.
func (w *Waffo) Webhook() *core.WebhookHandler {
	return core.NewWebhookHandler(w.config)
}

// GetConfig returns the SDK configuration.
func (w *Waffo) GetConfig() *config.WaffoConfig {
	return w.config
}

// GenerateKeyPair generates a new RSA-2048 key pair for testing or key rotation.
// Returns the key pair with Base64-encoded keys (PKCS#8 for private, X.509 for public).
func GenerateKeyPair() (*utils.KeyPair, error) {
	return utils.GenerateKeyPair()
}

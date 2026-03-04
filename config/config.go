package config

import (
	"os"

	"github.com/waffo-com/waffo-go/errors"
	"github.com/waffo-com/waffo-go/net"
	"github.com/waffo-com/waffo-go/utils"
)

// SDK version
const SDKVersion = "1.2.2"

// WaffoConfig contains the configuration for the Waffo SDK.
type WaffoConfig struct {
	// APIKey is the merchant's API key (required).
	APIKey string

	// PrivateKey is the merchant's private key in Base64 PKCS#8 format (required).
	PrivateKey string

	// WaffoPublicKey is Waffo's public key in Base64 X.509 format (required).
	WaffoPublicKey string

	// Environment is the API environment (SANDBOX or PRODUCTION).
	// Default: SANDBOX
	Environment Environment

	// MerchantID is the merchant ID for auto-injection.
	// If set, it will be automatically injected into requests.
	MerchantID string

	// ConnectTimeout is the TCP connection timeout in milliseconds.
	// Default: 10000 (10 seconds)
	ConnectTimeout int64

	// ReadTimeout is the response read timeout in milliseconds.
	// Default: 30000 (30 seconds)
	ReadTimeout int64

	// CustomTransport is a custom HTTP transport.
	// If nil, the default transport is used.
	CustomTransport net.HttpTransport
}

// ConfigBuilder is a builder for WaffoConfig.
type ConfigBuilder struct {
	config WaffoConfig
}

// NewConfigBuilder creates a new ConfigBuilder with default values.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: WaffoConfig{
			Environment:    Sandbox,
			ConnectTimeout: net.DefaultConnectTimeout,
			ReadTimeout:    net.DefaultReadTimeout,
		},
	}
}

// APIKey sets the API key.
func (b *ConfigBuilder) APIKey(apiKey string) *ConfigBuilder {
	b.config.APIKey = apiKey
	return b
}

// PrivateKey sets the merchant's private key.
func (b *ConfigBuilder) PrivateKey(privateKey string) *ConfigBuilder {
	b.config.PrivateKey = privateKey
	return b
}

// WaffoPublicKey sets Waffo's public key.
func (b *ConfigBuilder) WaffoPublicKey(publicKey string) *ConfigBuilder {
	b.config.WaffoPublicKey = publicKey
	return b
}

// Environment sets the API environment.
func (b *ConfigBuilder) Environment(env Environment) *ConfigBuilder {
	b.config.Environment = env
	return b
}

// MerchantID sets the merchant ID for auto-injection.
func (b *ConfigBuilder) MerchantID(merchantID string) *ConfigBuilder {
	b.config.MerchantID = merchantID
	return b
}

// ConnectTimeout sets the TCP connection timeout in milliseconds.
func (b *ConfigBuilder) ConnectTimeout(timeout int64) *ConfigBuilder {
	b.config.ConnectTimeout = timeout
	return b
}

// ReadTimeout sets the response read timeout in milliseconds.
func (b *ConfigBuilder) ReadTimeout(timeout int64) *ConfigBuilder {
	b.config.ReadTimeout = timeout
	return b
}

// CustomTransport sets a custom HTTP transport.
func (b *ConfigBuilder) CustomTransport(transport net.HttpTransport) *ConfigBuilder {
	b.config.CustomTransport = transport
	return b
}

// Build validates and returns the WaffoConfig.
func (b *ConfigBuilder) Build() (*WaffoConfig, error) {
	config := b.config

	// Validate required fields
	if config.APIKey == "" {
		return nil, errors.NewWaffoError(errors.CodeUnexpectedError, "API key is required")
	}
	if config.PrivateKey == "" {
		return nil, errors.NewWaffoError(errors.CodeInvalidPrivateKey, "private key is required")
	}
	if config.WaffoPublicKey == "" {
		return nil, errors.NewWaffoError(errors.CodeInvalidPublicKey, "Waffo public key is required")
	}

	// Validate private key format
	if err := utils.ValidatePrivateKey(config.PrivateKey); err != nil {
		return nil, err
	}

	// Validate public key format
	if err := utils.ValidatePublicKey(config.WaffoPublicKey); err != nil {
		return nil, err
	}

	// Set default values
	if config.Environment == "" {
		config.Environment = Sandbox
	}
	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = net.DefaultConnectTimeout
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = net.DefaultReadTimeout
	}

	return &config, nil
}

// MustBuild is like Build but panics on error.
func (b *ConfigBuilder) MustBuild() *WaffoConfig {
	config, err := b.Build()
	if err != nil {
		panic(err)
	}
	return config
}

// FromEnv creates a WaffoConfig from environment variables.
//
// Environment variables:
//   - WAFFO_API_KEY: API key (required)
//   - WAFFO_PRIVATE_KEY: Private key in Base64 (required)
//   - WAFFO_PUBLIC_KEY: Waffo's public key in Base64 (required)
//   - WAFFO_ENVIRONMENT: "SANDBOX" or "PRODUCTION" (default: SANDBOX)
//   - WAFFO_MERCHANT_ID: Merchant ID for auto-injection (optional)
func FromEnv() (*WaffoConfig, error) {
	builder := NewConfigBuilder().
		APIKey(os.Getenv("WAFFO_API_KEY")).
		PrivateKey(os.Getenv("WAFFO_PRIVATE_KEY")).
		WaffoPublicKey(os.Getenv("WAFFO_PUBLIC_KEY"))

	env := os.Getenv("WAFFO_ENVIRONMENT")
	if env == "PRODUCTION" {
		builder.Environment(Production)
	} else {
		builder.Environment(Sandbox)
	}

	merchantID := os.Getenv("WAFFO_MERCHANT_ID")
	if merchantID != "" {
		builder.MerchantID(merchantID)
	}

	return builder.Build()
}

// GetBaseURL returns the base URL for the configured environment.
func (c *WaffoConfig) GetBaseURL() string {
	return c.Environment.BaseURL()
}

// GetSDKVersion returns the SDK version string.
func (c *WaffoConfig) GetSDKVersion() string {
	return "waffo-go/" + SDKVersion
}

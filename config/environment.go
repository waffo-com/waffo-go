// Package config provides configuration types for the Waffo SDK.
package config

// Environment represents the API environment.
type Environment string

const (
	// Sandbox is the sandbox/test environment.
	Sandbox Environment = "SANDBOX"

	// Production is the production environment.
	Production Environment = "PRODUCTION"
)

// BaseURL returns the base URL for the environment.
func (e Environment) BaseURL() string {
	switch e {
	case Production:
		return "https://api.waffo.com/api/v1"
	case Sandbox:
		return "https://api-sandbox.waffo.com/api/v1"
	default:
		return "https://api-sandbox.waffo.com/api/v1"
	}
}

// String returns the string representation of the environment.
func (e Environment) String() string {
	return string(e)
}

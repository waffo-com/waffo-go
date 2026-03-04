//go:build e2e
// +build e2e

package e2e

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	waffo "github.com/waffo-com/waffo-go"
	"github.com/waffo-com/waffo-go/config"
)

// E2ETestConfig holds the configuration for E2E tests
// Configuration is read from test/e2e/application-test.yml
type E2ETestConfig struct {
	APIKey         string
	PrivateKey     string
	WaffoPublicKey string
	MerchantID     string
	Environment    string
}

var (
	instance *E2ETestConfig
	once     sync.Once
)

// GetInstance returns the singleton E2ETestConfig instance
func GetInstance() *E2ETestConfig {
	once.Do(func() {
		instance = &E2ETestConfig{}
		instance.loadConfig()
	})
	return instance
}

// loadConfig loads configuration from application-test.yml
func (c *E2ETestConfig) loadConfig() {
	// Try multiple paths for config file
	configPaths := []string{
		"application-test.yml",
		"test/e2e/application-test.yml",
		"../e2e/application-test.yml",
		"../../test/e2e/application-test.yml",
	}

	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile == "" {
		// Try to find from WAFFO_E2E_CONFIG environment variable
		if envPath := os.Getenv("WAFFO_E2E_CONFIG"); envPath != "" {
			configFile = envPath
		}
	}

	if configFile == "" {
		return // Config not found, IsConfigured() will return false
	}

	file, err := os.Open(configFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if strings.HasPrefix(line, "api-key:") {
			c.APIKey = extractValue(line)
		} else if strings.HasPrefix(line, "private-key:") {
			c.PrivateKey = extractValue(line)
		} else if strings.HasPrefix(line, "waffo-public-key:") {
			c.WaffoPublicKey = extractValue(line)
		} else if strings.HasPrefix(line, "merchant-id:") {
			c.MerchantID = extractValue(line)
		} else if strings.HasPrefix(line, "environment:") {
			c.Environment = extractValue(line)
		}
	}
}

func extractValue(line string) string {
	idx := strings.Index(line, ":")
	if idx > 0 && idx < len(line)-1 {
		return strings.TrimSpace(line[idx+1:])
	}
	return ""
}

// IsConfigured returns true if all required config values are present
func (c *E2ETestConfig) IsConfigured() bool {
	return c.APIKey != "" && c.PrivateKey != "" && c.WaffoPublicKey != ""
}

// CreateWaffoClient creates a Waffo client with the loaded configuration
func (c *E2ETestConfig) CreateWaffoClient() (*waffo.Waffo, error) {
	if !c.IsConfigured() {
		return nil, &ConfigError{Message: "Waffo config not properly configured"}
	}

	env := config.Sandbox
	if strings.ToUpper(c.Environment) == "PRODUCTION" {
		env = config.Production
	}

	cfg, err := config.NewConfigBuilder().
		APIKey(c.APIKey).
		PrivateKey(c.PrivateKey).
		WaffoPublicKey(c.WaffoPublicKey).
		MerchantID(c.MerchantID).
		Environment(env).
		Build()
	if err != nil {
		return nil, err
	}

	return waffo.New(cfg), nil
}

// GetConfigFilePath returns the expected config file path
func GetConfigFilePath() string {
	// Get the directory of the test file
	wd, _ := os.Getwd()
	return filepath.Join(wd, "test", "e2e", "application-test.yml")
}

// ConfigError represents a configuration error
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

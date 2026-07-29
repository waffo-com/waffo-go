package waffo

import (
	"strings"
	"testing"

	"github.com/waffo-com/waffo-go/v2/config"
	"github.com/waffo-com/waffo-go/v2/errors"
	"github.com/waffo-com/waffo-go/v2/utils"
)

func TestNewPanicsForInvalidPrivateKeyConfig(t *testing.T) {
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("New() should panic for invalid private key config")
		}
		waffoErr, ok := recovered.(*errors.WaffoError)
		if !ok {
			t.Fatalf("expected WaffoError panic, got %T", recovered)
		}
		if waffoErr.ErrorCode != errors.CodeInvalidPrivateKey {
			t.Fatalf("expected error code %s, got %s", errors.CodeInvalidPrivateKey, waffoErr.ErrorCode)
		}
		if waffoErr.Message != "Invalid private key format. Please provide an unencrypted PKCS#8 private key." {
			t.Fatalf("unexpected error message: %s", waffoErr.Message)
		}
		if strings.Contains(strings.ToLower(waffoErr.Error()), "wrong tag") {
			t.Fatalf("error should not expose wrong tag detail: %v", waffoErr)
		}
	}()

	New(&config.WaffoConfig{
		APIKey:         "test-api-key",
		PrivateKey:     "not-a-key",
		WaffoPublicKey: keyPair.PublicKey,
		Environment:    config.Sandbox,
		MerchantID:     "test-merchant-id",
	})
}

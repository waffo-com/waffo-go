package test

import (
	"testing"

	"github.com/waffo-com/waffo-go/errors"
	"github.com/waffo-com/waffo-go/utils"
)

// getTestKeyPair generates a fresh key pair for testing
// We use generated keys because test-vectors/rsa-signing.json contains placeholder keys
func getTestKeyPair(t *testing.T) *utils.KeyPair {
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}
	return keyPair
}

func TestSignAndVerify(t *testing.T) {
	keyPair := getTestKeyPair(t)

	testCases := []struct {
		name string
		data string
	}{
		{"basic_json", `{"paymentRequestId":"test-123","amount":"100.00"}`},
		{"unicode_data", `{"description":"测试订单","amount":"100.00"}`},
		{"empty_object", `{}`},
		{"complex_nested", `{"order":{"id":"123","items":[{"name":"Product","qty":1}]},"user":{"email":"test@example.com"}}`},
		{"special_characters", `{"desc":"Test & Demo <script>alert('xss')</script>","amount":"99.99"}`},
		{"empty_string", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Sign
			signature, err := utils.Sign(tc.data, keyPair.PrivateKey)
			if err != nil {
				t.Fatalf("Sign() failed: %v", err)
			}

			if signature == "" {
				t.Fatal("Sign() returned empty signature")
			}

			// Verify with correct public key
			valid := utils.Verify(tc.data, signature, keyPair.PublicKey)
			if !valid {
				t.Error("Verify() should return true for valid signature")
			}

			// Verify with tampered data
			tamperedData := tc.data + "x"
			valid = utils.Verify(tamperedData, signature, keyPair.PublicKey)
			if valid {
				t.Error("Verify() should return false for tampered data")
			}
		})
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"test":"data"}`

	// Invalid Base64 signature
	valid := utils.Verify(data, "!!!invalid-base64!!!", keyPair.PublicKey)
	if valid {
		t.Error("Verify() should return false for invalid Base64 signature")
	}

	// Valid Base64 but not a valid signature
	valid = utils.Verify(data, "aW52YWxpZC1zaWduYXR1cmU=", keyPair.PublicKey)
	if valid {
		t.Error("Verify() should return false for invalid signature")
	}
}

func TestValidatePrivateKey(t *testing.T) {
	keyPair := getTestKeyPair(t)

	// Valid key
	err := utils.ValidatePrivateKey(keyPair.PrivateKey)
	if err != nil {
		t.Errorf("ValidatePrivateKey() should succeed for valid key: %v", err)
	}

	// Empty key
	err = utils.ValidatePrivateKey("")
	if err == nil {
		t.Error("ValidatePrivateKey() should fail for empty key")
	}

	// Invalid key
	err = utils.ValidatePrivateKey("NOT_A_VALID_KEY")
	if err == nil {
		t.Error("ValidatePrivateKey() should fail for invalid key")
	}

	// Check error code
	if waffoErr, ok := err.(*errors.WaffoError); ok {
		if waffoErr.ErrorCode != errors.CodeInvalidPrivateKey {
			t.Errorf("Expected error code %s, got %s", errors.CodeInvalidPrivateKey, waffoErr.ErrorCode)
		}
	}
}

func TestValidatePublicKey(t *testing.T) {
	keyPair := getTestKeyPair(t)

	// Valid key
	err := utils.ValidatePublicKey(keyPair.PublicKey)
	if err != nil {
		t.Errorf("ValidatePublicKey() should succeed for valid key: %v", err)
	}

	// Empty key
	err = utils.ValidatePublicKey("")
	if err == nil {
		t.Error("ValidatePublicKey() should fail for empty key")
	}

	// Invalid key
	err = utils.ValidatePublicKey("NOT_A_VALID_KEY")
	if err == nil {
		t.Error("ValidatePublicKey() should fail for invalid key")
	}

	// Check error code
	if waffoErr, ok := err.(*errors.WaffoError); ok {
		if waffoErr.ErrorCode != errors.CodeInvalidPublicKey {
			t.Errorf("Expected error code %s, got %s", errors.CodeInvalidPublicKey, waffoErr.ErrorCode)
		}
	}
}

func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}

	if keyPair.PrivateKey == "" {
		t.Error("GenerateKeyPair() returned empty private key")
	}
	if keyPair.PublicKey == "" {
		t.Error("GenerateKeyPair() returned empty public key")
	}

	// Validate generated keys
	if err := utils.ValidatePrivateKey(keyPair.PrivateKey); err != nil {
		t.Errorf("Generated private key is invalid: %v", err)
	}
	if err := utils.ValidatePublicKey(keyPair.PublicKey); err != nil {
		t.Errorf("Generated public key is invalid: %v", err)
	}

	// Test sign/verify with generated keys
	data := `{"test":"generated-key-test"}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() with generated key failed: %v", err)
	}

	valid := utils.Verify(data, signature, keyPair.PublicKey)
	if !valid {
		t.Error("Verify() should return true for signature with generated key pair")
	}
}

func TestCrossKeyVerification(t *testing.T) {
	// Generate two different key pairs
	keyPair1, _ := utils.GenerateKeyPair()
	keyPair2, _ := utils.GenerateKeyPair()

	data := `{"test":"cross-key-test"}`

	// Sign with key pair 1
	signature, _ := utils.Sign(data, keyPair1.PrivateKey)

	// Verify with key pair 1 (should succeed)
	if !utils.Verify(data, signature, keyPair1.PublicKey) {
		t.Error("Verify() should return true with matching key pair")
	}

	// Verify with key pair 2 (should fail)
	if utils.Verify(data, signature, keyPair2.PublicKey) {
		t.Error("Verify() should return false with wrong public key")
	}
}

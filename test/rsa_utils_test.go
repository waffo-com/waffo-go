package test

import (
	"strings"
	"testing"

	"github.com/waffo-com/waffo-go/errors"
	"github.com/waffo-com/waffo-go/utils"
)

const friendlyPrivateKeyMessage = "Invalid private key format. Please provide an unencrypted PKCS#8 private key."
const pkcs1PrivateKeyPEM = "-----BEGIN RSA PRIVATE KEY-----\n" +
	"bm90LWEta2V5\n" +
	"-----END RSA PRIVATE KEY-----"
const encryptedPrivateKeyPEM = "-----BEGIN ENCRYPTED PRIVATE KEY-----\n" +
	"bm90LWEta2V5\n" +
	"-----END ENCRYPTED PRIVATE KEY-----"

// getTestKeyPair generates a fresh key pair for testing
// We use generated keys because test-vectors/rsa-signing.json contains placeholder keys
func getTestKeyPair(t *testing.T) *utils.KeyPair {
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}
	return keyPair
}

func toPKCS8PrivateKeyPEM(base64PrivateKey string) string {
	var builder strings.Builder
	builder.WriteString("-----BEGIN PRIVATE KEY-----\n")
	for len(base64PrivateKey) > 64 {
		builder.WriteString(base64PrivateKey[:64])
		builder.WriteByte('\n')
		base64PrivateKey = base64PrivateKey[64:]
	}
	builder.WriteString(base64PrivateKey)
	builder.WriteString("\n-----END PRIVATE KEY-----")
	return builder.String()
}

func assertFriendlyPrivateKeyError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected private key validation to fail")
	}
	waffoErr, ok := err.(*errors.WaffoError)
	if !ok {
		t.Fatalf("expected WaffoError, got %T", err)
	}
	if waffoErr.ErrorCode != errors.CodeInvalidPrivateKey {
		t.Fatalf("expected error code %s, got %s", errors.CodeInvalidPrivateKey, waffoErr.ErrorCode)
	}
	if waffoErr.Message != friendlyPrivateKeyMessage {
		t.Fatalf("expected friendly message %q, got %q", friendlyPrivateKeyMessage, waffoErr.Message)
	}
	if strings.Contains(strings.ToLower(err.Error()), "wrong tag") {
		t.Fatalf("error should not expose wrong tag detail: %v", err)
	}
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

	// PKCS#8 PEM key
	pemKey := toPKCS8PrivateKeyPEM(keyPair.PrivateKey)
	if err := utils.ValidatePrivateKey(pemKey); err != nil {
		t.Errorf("ValidatePrivateKey() should succeed for PKCS#8 PEM: %v", err)
	}

	signature, err := utils.Sign(`{"test":"pem"}`, pemKey)
	if err != nil {
		t.Fatalf("Sign() should succeed for PKCS#8 PEM: %v", err)
	}
	if !utils.Verify(`{"test":"pem"}`, signature, keyPair.PublicKey) {
		t.Fatal("Verify() should succeed for signature created with PKCS#8 PEM")
	}

	assertFriendlyPrivateKeyError(t, utils.ValidatePrivateKey(pkcs1PrivateKeyPEM))
	assertFriendlyPrivateKeyError(t, utils.ValidatePrivateKey(encryptedPrivateKeyPEM))
	assertFriendlyPrivateKeyError(t, utils.ValidatePrivateKey("not-a-key"))

	_, err = utils.Sign("{}", pkcs1PrivateKeyPEM)
	assertFriendlyPrivateKeyError(t, err)
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

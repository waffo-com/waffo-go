package vectors

import (
	"testing"

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

// RSA-001: Sign basic JSON payload
func TestRSA001_SignBasicJSON(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"paymentRequestId":"test-123","amount":"100.00"}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if signature == "" {
		t.Fatal("Sign() returned empty signature")
	}
	// Cross-verify with the same key pair
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for valid signature")
	}
}

// RSA-002: Sign JSON with Unicode characters (UTF-8)
func TestRSA002_SignUnicodeData(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"description":"测试订单","amount":"100.00"}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for Unicode data signature")
	}
}

// RSA-003: Sign empty JSON object
func TestRSA003_SignEmptyObject(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for empty object signature")
	}
}

// RSA-004: Sign complex nested JSON
func TestRSA004_SignComplexNested(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"order":{"id":"123","items":[{"name":"Product","qty":1}]},"user":{"email":"test@example.com"}}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for complex nested signature")
	}
}

// RSA-005: Sign JSON with special characters
func TestRSA005_SignSpecialCharacters(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"desc":"Test & Demo <script>alert('xss')</script>","amount":"99.99"}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for special characters signature")
	}
}

// RSA-006: Sign large JSON payload
func TestRSA006_SignLargePayload(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"items":[{"id":1,"name":"Product 1","price":"10.00"},{"id":2,"name":"Product 2","price":"20.00"},{"id":3,"name":"Product 3","price":"30.00"}],"metadata":{"version":"1.0","timestamp":"2024-01-01T00:00:00Z"}}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for large payload signature")
	}
}

// RSA-007: Sign empty string
func TestRSA007_SignEmptyString(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := ""
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for empty string signature")
	}
}

// RSA-V02: Verify invalid signature returns false
func TestRSAV02_VerifyInvalidSignature(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"test":"data"}`
	// Invalid but valid Base64 signature
	invalidSignature := "aW52YWxpZC1zaWduYXR1cmU="
	if utils.Verify(data, invalidSignature, keyPair.PublicKey) {
		t.Error("Verify() should return false for invalid signature")
	}
}

// RSA-V03: Verify signature fails if data is tampered
func TestRSAV03_VerifyTamperedData(t *testing.T) {
	keyPair := getTestKeyPair(t)
	originalData := `{"test":"original"}`
	tamperedData := `{"test":"tampered"}`

	signature, err := utils.Sign(originalData, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Verify original data succeeds
	if !utils.Verify(originalData, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true for original data")
	}

	// Verify tampered data fails
	if utils.Verify(tamperedData, signature, keyPair.PublicKey) {
		t.Error("Verify() should return false for tampered data")
	}
}

// RSA-V04: Verify signature fails with wrong public key
func TestRSAV04_VerifyWrongPublicKey(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"test":"data"}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Generate different key pair
	wrongKeyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}

	// Verify with wrong public key should fail
	if utils.Verify(data, signature, wrongKeyPair.PublicKey) {
		t.Error("Verify() should return false with wrong public key")
	}
}

// RSA-E01: Invalid private key throws error
func TestRSAE01_InvalidPrivateKeyFormat(t *testing.T) {
	data := `{"test":"data"}`
	_, err := utils.Sign(data, "NOT_A_VALID_KEY")
	if err == nil {
		t.Error("Sign() should return error for invalid private key")
	}
}

// RSA-E02: Invalid public key returns false (not error)
func TestRSAE02_InvalidPublicKeyFormat(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"test":"data"}`
	signature, _ := utils.Sign(data, keyPair.PrivateKey)
	// Verify with invalid public key should return false
	if utils.Verify(data, signature, "NOT_A_VALID_KEY") {
		t.Error("Verify() should return false for invalid public key")
	}
}

// RSA-E04: Corrupted Base64 signature handled gracefully
func TestRSAE04_CorruptedSignature(t *testing.T) {
	keyPair := getTestKeyPair(t)
	data := `{"test":"data"}`
	// Invalid Base64 signature
	if utils.Verify(data, "!!!invalid-base64!!!", keyPair.PublicKey) {
		t.Error("Verify() should return false for corrupted Base64 signature")
	}
}

// Test key pair generation matches specification
func TestKeyPairGeneration(t *testing.T) {
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}

	// Verify keys are valid
	if err := utils.ValidatePrivateKey(keyPair.PrivateKey); err != nil {
		t.Errorf("Generated private key is invalid: %v", err)
	}
	if err := utils.ValidatePublicKey(keyPair.PublicKey); err != nil {
		t.Errorf("Generated public key is invalid: %v", err)
	}

	// Test sign/verify with generated keys
	data := `{"test":"keypair-generation"}`
	signature, err := utils.Sign(data, keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() with generated key failed: %v", err)
	}
	if !utils.Verify(data, signature, keyPair.PublicKey) {
		t.Error("Verify() should return true with generated key pair")
	}
}

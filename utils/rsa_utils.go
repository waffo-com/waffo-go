// Package utils provides utility functions for the Waffo SDK.
package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/waffo-com/waffo-go/errors"
)

// KeyPair represents a RSA key pair with Base64 encoded keys.
type KeyPair struct {
	PrivateKey string // PKCS#8 DER encoded, Base64 string
	PublicKey  string // X.509 SubjectPublicKeyInfo DER encoded, Base64 string
}

// Sign signs the data with the given Base64-encoded PKCS#8 private key.
// Returns the Base64-encoded signature.
//
// Algorithm: SHA256withRSA (RSASSA-PKCS1-v1_5)
func Sign(data string, base64PrivateKey string) (string, error) {
	// Decode private key from Base64
	keyBytes, err := base64.StdEncoding.DecodeString(base64PrivateKey)
	if err != nil {
		return "", errors.NewWaffoErrorWithCause(errors.CodeSigningFailed, "failed to decode private key", err)
	}

	// Parse PKCS#8 private key
	privateKey, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return "", errors.NewWaffoErrorWithCause(errors.CodeSigningFailed, "failed to parse private key", err)
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.NewWaffoError(errors.CodeSigningFailed, "private key is not RSA key")
	}

	// Hash the data with SHA256
	hashed := sha256.Sum256([]byte(data))

	// Sign with PKCS1v15
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", errors.NewWaffoErrorWithCause(errors.CodeSigningFailed, "failed to sign data", err)
	}

	// Return Base64 encoded signature
	return base64.StdEncoding.EncodeToString(signature), nil
}

// Verify verifies the signature against the data using the given Base64-encoded X.509 public key.
// Returns true if the signature is valid, false otherwise.
//
// Algorithm: SHA256withRSA (RSASSA-PKCS1-v1_5)
func Verify(data string, base64Signature string, base64PublicKey string) bool {
	// Decode public key from Base64
	keyBytes, err := base64.StdEncoding.DecodeString(base64PublicKey)
	if err != nil {
		return false
	}

	// Parse X.509 public key
	publicKey, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return false
	}

	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return false
	}

	// Decode signature from Base64
	signatureBytes, err := base64.StdEncoding.DecodeString(base64Signature)
	if err != nil {
		return false
	}

	// Hash the data with SHA256
	hashed := sha256.Sum256([]byte(data))

	// Verify with PKCS1v15
	err = rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, hashed[:], signatureBytes)
	return err == nil
}

// ValidatePrivateKey validates the given Base64-encoded PKCS#8 private key.
// Returns an error if the key is invalid.
func ValidatePrivateKey(base64PrivateKey string) error {
	if base64PrivateKey == "" {
		return errors.NewWaffoError(errors.CodeInvalidPrivateKey, "private key is null or empty")
	}

	// Decode from Base64
	keyBytes, err := base64.StdEncoding.DecodeString(base64PrivateKey)
	if err != nil {
		return errors.NewWaffoErrorWithCause(errors.CodeInvalidPrivateKey, "invalid private key: failed to decode base64", err)
	}

	// Parse PKCS#8 private key
	privateKey, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return errors.NewWaffoErrorWithCause(errors.CodeInvalidPrivateKey, "invalid private key: failed to parse PKCS#8", err)
	}

	// Ensure it's an RSA key
	if _, ok := privateKey.(*rsa.PrivateKey); !ok {
		return errors.NewWaffoError(errors.CodeInvalidPrivateKey, "invalid private key: not an RSA key")
	}

	return nil
}

// ValidatePublicKey validates the given Base64-encoded X.509 public key.
// Returns an error if the key is invalid.
func ValidatePublicKey(base64PublicKey string) error {
	if base64PublicKey == "" {
		return errors.NewWaffoError(errors.CodeInvalidPublicKey, "public key is null or empty")
	}

	// Decode from Base64
	keyBytes, err := base64.StdEncoding.DecodeString(base64PublicKey)
	if err != nil {
		return errors.NewWaffoErrorWithCause(errors.CodeInvalidPublicKey, "invalid public key: failed to decode base64", err)
	}

	// Parse X.509 public key
	publicKey, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return errors.NewWaffoErrorWithCause(errors.CodeInvalidPublicKey, "invalid public key: failed to parse X.509", err)
	}

	// Ensure it's an RSA key
	if _, ok := publicKey.(*rsa.PublicKey); !ok {
		return errors.NewWaffoError(errors.CodeInvalidPublicKey, "invalid public key: not an RSA key")
	}

	return nil
}

// GenerateKeyPair generates a new RSA-2048 key pair.
// Returns the key pair with Base64-encoded keys (PKCS#8 for private, X.509 for public).
func GenerateKeyPair() (*KeyPair, error) {
	// Generate RSA-2048 key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to generate key pair", err)
	}

	// Export private key as PKCS#8 DER
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to marshal private key", err)
	}

	// Export public key as X.509 SPKI DER
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, errors.NewWaffoErrorWithCause(errors.CodeUnexpectedError, "failed to marshal public key", err)
	}

	return &KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(privateKeyBytes),
		PublicKey:  base64.StdEncoding.EncodeToString(publicKeyBytes),
	}, nil
}

// MustSign is like Sign but panics on error.
// Only use this in tests or when you are certain the key is valid.
func MustSign(data string, base64PrivateKey string) string {
	signature, err := Sign(data, base64PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("MustSign failed: %v", err))
	}
	return signature
}

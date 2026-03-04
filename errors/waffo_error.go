// Package errors provides error types for the Waffo SDK.
package errors

import "fmt"

// Error codes for WaffoError
const (
	CodeInvalidPublicKey     = "S0002" // Invalid public key
	CodeSigningFailed        = "S0003" // Failed to sign data
	CodeVerificationFailed   = "S0004" // Response signature verification failed
	CodeSerializationFailed  = "S0005" // Request serialization failed
	CodeUnexpectedError      = "S0006" // Unexpected error
	CodeInvalidPrivateKey    = "S0007" // Invalid private key
)

// WaffoError is the base error class for client-side errors
// (configuration errors, serialization failures, etc.).
type WaffoError struct {
	ErrorCode string
	Message   string
	Cause     error
}

// NewWaffoError creates a new WaffoError with the given error code and message.
func NewWaffoError(errorCode, message string) *WaffoError {
	return &WaffoError{
		ErrorCode: errorCode,
		Message:   message,
	}
}

// NewWaffoErrorWithCause creates a new WaffoError with the given error code, message, and cause.
func NewWaffoErrorWithCause(errorCode, message string, cause error) *WaffoError {
	return &WaffoError{
		ErrorCode: errorCode,
		Message:   message,
		Cause:     cause,
	}
}

// Error implements the error interface.
func (e *WaffoError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.ErrorCode, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.ErrorCode, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *WaffoError) Unwrap() error {
	return e.Cause
}

// Is checks if the target error is a WaffoError with the same error code.
func (e *WaffoError) Is(target error) bool {
	t, ok := target.(*WaffoError)
	if !ok {
		return false
	}
	return e.ErrorCode == t.ErrorCode
}

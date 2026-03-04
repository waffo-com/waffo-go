package errors

import "fmt"

// Error codes for WaffoUnknownStatusError
const (
	CodeNetworkError  = "S0001" // Network error (timeout, connection failed)
	CodeUnknownStatus = "E0001" // Server returned unknown status
)

// WaffoUnknownStatusError is a special error for network errors or unknown payment status.
// This error indicates that the payment may have been processed but the result is unknown.
//
// IMPORTANT: When WaffoUnknownStatusError is thrown for a payment request,
// the merchant MUST NOT assume the payment failed. The merchant SHOULD use
// the inquiry API to check the actual payment status.
type WaffoUnknownStatusError struct {
	WaffoError
}

// NewWaffoUnknownStatusError creates a new WaffoUnknownStatusError with the given error code and message.
func NewWaffoUnknownStatusError(errorCode, message string) *WaffoUnknownStatusError {
	return &WaffoUnknownStatusError{
		WaffoError: WaffoError{
			ErrorCode: errorCode,
			Message:   message,
		},
	}
}

// NewWaffoUnknownStatusErrorWithCause creates a new WaffoUnknownStatusError with the given error code, message, and cause.
func NewWaffoUnknownStatusErrorWithCause(errorCode, message string, cause error) *WaffoUnknownStatusError {
	return &WaffoUnknownStatusError{
		WaffoError: WaffoError{
			ErrorCode: errorCode,
			Message:   message,
			Cause:     cause,
		},
	}
}

// NewNetworkError creates a new WaffoUnknownStatusError for network errors.
func NewNetworkError(message string, cause error) *WaffoUnknownStatusError {
	return NewWaffoUnknownStatusErrorWithCause(CodeNetworkError, message, cause)
}

// Error implements the error interface.
func (e *WaffoUnknownStatusError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.ErrorCode, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.ErrorCode, e.Message)
}

// Is checks if the target error is a WaffoUnknownStatusError with the same error code.
func (e *WaffoUnknownStatusError) Is(target error) bool {
	t, ok := target.(*WaffoUnknownStatusError)
	if !ok {
		return false
	}
	return e.ErrorCode == t.ErrorCode
}

// IsNetworkError checks if the error is a network error (S0001).
func (e *WaffoUnknownStatusError) IsNetworkError() bool {
	return e.ErrorCode == CodeNetworkError
}

// IsUnknownStatus checks if the error is an unknown status error (E0001).
func (e *WaffoUnknownStatusError) IsUnknownStatus() bool {
	return e.ErrorCode == CodeUnknownStatus
}

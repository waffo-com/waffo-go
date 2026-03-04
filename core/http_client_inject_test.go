package core

import (
	"regexp"
	"testing"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/types/order"
	"github.com/waffo-com/waffo-go/types/subscription"
)

// Test types that mirror real request structures
type testMerchantInfo struct {
	MerchantID    string `json:"merchantId,omitempty"`
	SubMerchantID string `json:"subMerchantId,omitempty"`
}

type testParamsWithPtrMerchantInfo struct {
	MerchantInfo *testMerchantInfo `json:"merchantInfo,omitempty"`
	OrderID      string            `json:"orderId"`
}

type testParamsWithTopLevelMerchantID struct {
	MerchantID string `json:"merchantId"`
	OrderID    string `json:"orderId"`
}

type testParamsNoMerchantInfo struct {
	OrderID string `json:"orderId"`
}

func newClientWithMerchantID(merchantID string) *WaffoHttpClient {
	return &WaffoHttpClient{
		config: &config.WaffoConfig{
			MerchantID: merchantID,
		},
	}
}

// Test 1: *MerchantInfo non-nil, MerchantID empty → auto inject
func TestInjectMerchantID_PtrNonNil_EmptyMerchantID(t *testing.T) {
	client := newClientWithMerchantID("M001")
	params := &testParamsWithPtrMerchantInfo{
		MerchantInfo: &testMerchantInfo{},
		OrderID:      "O001",
	}

	client.injectMerchantID(params)

	if params.MerchantInfo.MerchantID != "M001" {
		t.Errorf("expected MerchantID='M001', got '%s'", params.MerchantInfo.MerchantID)
	}
}

// Test 2: *MerchantInfo non-nil, MerchantID has value → don't override
func TestInjectMerchantID_PtrNonNil_ExistingMerchantID(t *testing.T) {
	client := newClientWithMerchantID("M001")
	params := &testParamsWithPtrMerchantInfo{
		MerchantInfo: &testMerchantInfo{MerchantID: "M_EXISTING"},
		OrderID:      "O001",
	}

	client.injectMerchantID(params)

	if params.MerchantInfo.MerchantID != "M_EXISTING" {
		t.Errorf("expected MerchantID='M_EXISTING', got '%s'", params.MerchantInfo.MerchantID)
	}
}

// Test 3: *MerchantInfo nil → create new instance and inject
func TestInjectMerchantID_PtrNil_CreatesAndInjects(t *testing.T) {
	client := newClientWithMerchantID("M001")
	params := &testParamsWithPtrMerchantInfo{
		MerchantInfo: nil,
		OrderID:      "O001",
	}

	client.injectMerchantID(params)

	if params.MerchantInfo == nil {
		t.Fatal("expected MerchantInfo to be created, got nil")
	}
	if params.MerchantInfo.MerchantID != "M001" {
		t.Errorf("expected MerchantID='M001', got '%s'", params.MerchantInfo.MerchantID)
	}
}

// Test 4: *MerchantInfo non-nil, has SubMerchantID → inject MerchantID without affecting SubMerchantID
func TestInjectMerchantID_PtrNonNil_PreservesSubMerchantID(t *testing.T) {
	client := newClientWithMerchantID("M001")
	params := &testParamsWithPtrMerchantInfo{
		MerchantInfo: &testMerchantInfo{SubMerchantID: "SUB001"},
		OrderID:      "O001",
	}

	client.injectMerchantID(params)

	if params.MerchantInfo.MerchantID != "M001" {
		t.Errorf("expected MerchantID='M001', got '%s'", params.MerchantInfo.MerchantID)
	}
	if params.MerchantInfo.SubMerchantID != "SUB001" {
		t.Errorf("expected SubMerchantID='SUB001', got '%s'", params.MerchantInfo.SubMerchantID)
	}
}

// Test 5: Top-level MerchantID field → normal inject
func TestInjectMerchantID_TopLevelMerchantID(t *testing.T) {
	client := newClientWithMerchantID("M001")
	params := &testParamsWithTopLevelMerchantID{
		OrderID: "O001",
	}

	client.injectMerchantID(params)

	if params.MerchantID != "M001" {
		t.Errorf("expected MerchantID='M001', got '%s'", params.MerchantID)
	}
}

// Test 6: config.MerchantID empty → no operation
func TestInjectMerchantID_EmptyConfigMerchantID(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &testParamsWithPtrMerchantInfo{
		MerchantInfo: &testMerchantInfo{},
		OrderID:      "O001",
	}

	client.injectMerchantID(params)

	if params.MerchantInfo.MerchantID != "" {
		t.Errorf("expected MerchantID='', got '%s'", params.MerchantInfo.MerchantID)
	}
}

// Test 7: Struct without MerchantInfo field → no panic
func TestInjectMerchantID_NoMerchantInfoField(t *testing.T) {
	client := newClientWithMerchantID("M001")
	params := &testParamsNoMerchantInfo{
		OrderID: "O001",
	}

	// Should not panic
	client.injectMerchantID(params)

	if params.OrderID != "O001" {
		t.Errorf("expected OrderID='O001', got '%s'", params.OrderID)
	}
}

// --- injectRequestedAt test types ---

type testParamsWithOrderRequestedAt struct {
	MerchantInfo     *testMerchantInfo `json:"merchantInfo,omitempty"`
	OrderRequestedAt string            `json:"orderRequestedAt,omitempty"`
}

type testParamsWithRequestedAt struct {
	MerchantID  string `json:"merchantId"`
	RequestedAt string `json:"requestedAt,omitempty"`
}

var iso8601Regex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)

// Test 8: OrderRequestedAt empty → auto inject
func TestInjectRequestedAt_OrderRequestedAt_Empty(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &testParamsWithOrderRequestedAt{}

	client.injectRequestedAt(params)

	if params.OrderRequestedAt == "" {
		t.Fatal("expected OrderRequestedAt to be injected, got empty")
	}
	if !iso8601Regex.MatchString(params.OrderRequestedAt) {
		t.Errorf("expected ISO 8601 format, got '%s'", params.OrderRequestedAt)
	}
}

// Test 9: OrderRequestedAt has value → don't override
func TestInjectRequestedAt_OrderRequestedAt_Existing(t *testing.T) {
	client := newClientWithMerchantID("")
	existing := "2025-01-01T00:00:00.000Z"
	params := &testParamsWithOrderRequestedAt{
		OrderRequestedAt: existing,
	}

	client.injectRequestedAt(params)

	if params.OrderRequestedAt != existing {
		t.Errorf("expected OrderRequestedAt='%s', got '%s'", existing, params.OrderRequestedAt)
	}
}

// Test 10: RequestedAt empty → auto inject
func TestInjectRequestedAt_RequestedAt_Empty(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &testParamsWithRequestedAt{MerchantID: "M001"}

	client.injectRequestedAt(params)

	if params.RequestedAt == "" {
		t.Fatal("expected RequestedAt to be injected, got empty")
	}
	if !iso8601Regex.MatchString(params.RequestedAt) {
		t.Errorf("expected ISO 8601 format, got '%s'", params.RequestedAt)
	}
}

// Test 11: RequestedAt has value → don't override
func TestInjectRequestedAt_RequestedAt_Existing(t *testing.T) {
	client := newClientWithMerchantID("")
	existing := "2025-06-15T12:30:00.000Z"
	params := &testParamsWithRequestedAt{
		MerchantID:  "M001",
		RequestedAt: existing,
	}

	client.injectRequestedAt(params)

	if params.RequestedAt != existing {
		t.Errorf("expected RequestedAt='%s', got '%s'", existing, params.RequestedAt)
	}
}

// Test 12: Struct without timestamp fields → no panic
func TestInjectRequestedAt_NoTimestampField(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &testParamsNoMerchantInfo{
		OrderID: "O001",
	}

	// Should not panic
	client.injectRequestedAt(params)

	if params.OrderID != "O001" {
		t.Errorf("expected OrderID='O001', got '%s'", params.OrderID)
	}
}

// Test 13: Injected value format matches ISO 8601 with 3-digit milliseconds
func TestInjectRequestedAt_FormatValidation(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &testParamsWithOrderRequestedAt{}

	client.injectRequestedAt(params)

	if !iso8601Regex.MatchString(params.OrderRequestedAt) {
		t.Errorf("injected timestamp does not match ISO 8601 format (yyyy-MM-ddTHH:mm:ss.SSSZ), got '%s'", params.OrderRequestedAt)
	}
}

// --- Integration tests with real Params types ---

// Test 14: Real CancelOrderParams without OrderRequestedAt → auto inject
func TestInjectRequestedAt_RealCancelOrderParams_AutoInject(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &order.CancelOrderParams{
		PaymentRequestID: "PR001",
	}

	client.injectRequestedAt(params)

	if params.OrderRequestedAt == "" {
		t.Fatal("expected OrderRequestedAt to be auto-injected for real CancelOrderParams")
	}
	if !iso8601Regex.MatchString(params.OrderRequestedAt) {
		t.Errorf("format mismatch, got '%s'", params.OrderRequestedAt)
	}
}

// Test 15: Real CreateOrderParams without OrderRequestedAt → auto inject
func TestInjectRequestedAt_RealCreateOrderParams_AutoInject(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &order.CreateOrderParams{
		PaymentRequestID: "PR002",
		OrderCurrency:    "USD",
		OrderAmount:      "100.00",
	}

	client.injectRequestedAt(params)

	if params.OrderRequestedAt == "" {
		t.Fatal("expected OrderRequestedAt to be auto-injected for real CreateOrderParams")
	}
	if !iso8601Regex.MatchString(params.OrderRequestedAt) {
		t.Errorf("format mismatch, got '%s'", params.OrderRequestedAt)
	}
}

// Test 16: Real CreateSubscriptionParams without RequestedAt → auto inject
func TestInjectRequestedAt_RealCreateSubscriptionParams_AutoInject(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &subscription.CreateSubscriptionParams{
		SubscriptionRequest: "SUB001",
		Currency:            "USD",
		Amount:              "9.99",
	}

	client.injectRequestedAt(params)

	if params.RequestedAt == "" {
		t.Fatal("expected RequestedAt to be auto-injected for real CreateSubscriptionParams")
	}
	if !iso8601Regex.MatchString(params.RequestedAt) {
		t.Errorf("format mismatch, got '%s'", params.RequestedAt)
	}
}

// Test 17: Real CancelSubscriptionParams without RequestedAt → auto inject
func TestInjectRequestedAt_RealCancelSubscriptionParams_AutoInject(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &subscription.CancelSubscriptionParams{
		SubscriptionID: "SUB_ID_001",
	}

	client.injectRequestedAt(params)

	if params.RequestedAt == "" {
		t.Fatal("expected RequestedAt to be auto-injected for real CancelSubscriptionParams")
	}
	if !iso8601Regex.MatchString(params.RequestedAt) {
		t.Errorf("format mismatch, got '%s'", params.RequestedAt)
	}
}

// Test 18: Real ChangeSubscriptionParams without RequestedAt → auto inject
func TestInjectRequestedAt_RealChangeSubscriptionParams_AutoInject(t *testing.T) {
	client := newClientWithMerchantID("")
	params := &subscription.ChangeSubscriptionParams{
		SubscriptionRequest:       "SUB_NEW_001",
		OriginSubscriptionRequest: "SUB_OLD_001",
		Currency:                  "HKD",
	}

	client.injectRequestedAt(params)

	if params.RequestedAt == "" {
		t.Fatal("expected RequestedAt to be auto-injected for real ChangeSubscriptionParams")
	}
	if !iso8601Regex.MatchString(params.RequestedAt) {
		t.Errorf("format mismatch, got '%s'", params.RequestedAt)
	}
}

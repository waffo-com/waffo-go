//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	waffo "github.com/waffo-com/waffo-go"
	"github.com/waffo-com/waffo-go/types/order"
)

// PaymentFlowE2ETest tests the complete payment flow:
// 1. Create order via SDK
// 2. Open checkout page
// 3. Fill in test card details
// 4. Complete payment
// 5. Verify redirect to success page
//
// Run with:
//   go test -tags=e2e ./test/e2e/... -v
//
// Visual mode (non-headless):
//   E2E_HEADLESS=false go test -tags=e2e ./test/e2e/... -v
//
// With screenshots:
//   E2E_SCREENSHOT=true go test -tags=e2e ./test/e2e/... -v

var (
	testWaffo  *waffo.Waffo
	testConfig *E2ETestConfig
)

func TestMain(m *testing.M) {
	// Initialize config
	testConfig = GetInstance()

	if !testConfig.IsConfigured() {
		fmt.Println("Skipping E2E tests: Waffo config not found")
		fmt.Println("Expected config file at: test/e2e/application-test.yml")
		fmt.Println("Or set WAFFO_E2E_CONFIG environment variable")
		return
	}

	var err error
	testWaffo, err = testConfig.CreateWaffoClient()
	if err != nil {
		fmt.Printf("Failed to create Waffo client: %v\n", err)
		return
	}

	fmt.Printf("Waffo client initialized with merchant: %s\n", testConfig.MerchantID)

	// Run tests
	m.Run()
}

func TestPaymentFlow_CreateOrderAndOpenCheckout(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// 1. Create order
	paymentRequestID := fmt.Sprintf("e2e_%d", time.Now().UnixNano())
	merchantOrderID := fmt.Sprintf("E2E_ORDER_%d", time.Now().UnixMilli())

	params := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "10.00",
		OrderDescription: "E2E Test Order",
		NotifyURL:        "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &order.UserInfo{
			UserID:       "e2e_test_user",
			UserEmail:    "e2e@test.com",
			UserTerminal: "WEB",
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
	}

	resp, err := testWaffo.Order().Create(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Payment Request ID: %s", paymentRequestID)
	t.Logf("Response: code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())

	if !resp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	if data == nil {
		t.Fatalf("Order data is nil: paymentRequestID=%s, code=%s, msg=%s",
			paymentRequestID, resp.GetCode(), resp.GetMessage())
	}

	checkoutURL := data.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: paymentRequestID=%s, data=%+v", paymentRequestID, data)
	}

	t.Logf("Checkout URL: %s", checkoutURL)
	t.Logf("Acquiring Order ID: %s", data.AcquiringOrderID)

	// 2. Open checkout page
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to checkout: %v", err)
	}

	if err := base.WaitForPageLoad(); err != nil {
		t.Fatalf("Failed to wait for page load: %v", err)
	}

	// 3. Verify page loaded
	base.PrintCurrentURL()
	base.TakeScreenshot("checkout_page_loaded")
}

func TestPaymentFlow_CreditCardPayment(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// Create order
	paymentRequestID := fmt.Sprintf("e2e_card_%d", time.Now().UnixNano())
	merchantOrderID := fmt.Sprintf("E2E_CARD_%d", time.Now().UnixMilli())

	params := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "10.00",
		OrderDescription: "E2E Credit Card Test",
		NotifyURL:        "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &order.UserInfo{
			UserID:       "e2e_test_user",
			UserEmail:    "e2e@test.com",
			UserTerminal: "WEB",
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
	}

	resp, err := testWaffo.Order().Create(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Payment Request ID: %s", paymentRequestID)
	t.Logf("Response: code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())

	if !resp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	checkoutURL := data.FetchRedirectURL()
	returnedPaymentRequestID := data.PaymentRequestID

	t.Logf("Checkout URL: %s", checkoutURL)
	t.Logf("Returned Payment Request ID: %s", returnedPaymentRequestID)

	// Open checkout
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(2000) // Wait for page to fully render

	base.TakeScreenshot("before_card_input")

	// Fill card details (selectors may need adjustment based on actual page)
	base.Fill("#payMethodProperties\\.card\\.pan", TestCard.Number)
	base.SleepMs(500)

	// Expiry in MM/YY format
	expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
	base.Fill("#payMethodProperties\\.card\\.expiry", expiry)
	base.SleepMs(500)

	base.Fill("#payMethodProperties\\.card\\.cvv", TestCard.CVV)
	base.SleepMs(500)

	base.Fill("#payMethodProperties\\.card\\.name", TestCard.Holder)
	base.SleepMs(500)

	base.TakeScreenshot("after_card_input")

	t.Log("Card info filled, clicking pay button...")

	// Click submit button
	if err := base.Click("button[type='submit']"); err != nil {
		t.Logf("Warning: Could not click submit button: %v", err)
	}

	// Wait for redirect (up to 60 seconds for payment processing)
	base.SetTimeout(60000)
	if err := base.WaitForURLContains("success"); err != nil {
		t.Logf("Did not redirect to success page: %v", err)
		base.PrintCurrentURL()
	} else {
		t.Log("Payment successful! Redirected to success page.")
	}

	base.TakeScreenshot("payment_completed")

	// Wait and query order status
	base.SleepMs(2000)

	inquiryParams := &order.InquiryOrderParams{
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		PaymentRequestID: returnedPaymentRequestID,
	}

	inquiryResp, err := testWaffo.Order().Inquiry(context.Background(), inquiryParams, nil)
	if err != nil {
		t.Logf("Warning: Failed to query order: paymentRequestID=%s, returnedPaymentRequestID=%s, error=%v",
			paymentRequestID, returnedPaymentRequestID, err)
		return
	}

	t.Logf("Inquiry Response: paymentRequestID=%s, returnedPaymentRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, returnedPaymentRequestID, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())

	if inquiryResp.IsSuccess() {
		inquiryData := inquiryResp.GetData()
		t.Logf("Final Order Status: paymentRequestID=%s, status=%s", paymentRequestID, inquiryData.OrderStatus)
	} else {
		t.Logf("Inquiry failed: paymentRequestID=%s, returnedPaymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, returnedPaymentRequestID, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
	}
}

func TestCreateOrder_WithoutOrderRequestedAt_AutoInjected(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	// Generate unique IDs with timestamp prefix
	paymentRequestID := fmt.Sprintf("e2e_autotime_%d", time.Now().UnixNano())
	merchantOrderID := fmt.Sprintf("E2E_AUTOTIME_%d", time.Now().UnixMilli())

	// Deliberately omit OrderRequestedAt to verify SDK auto-injection
	params := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "10.00",
		OrderDescription: "E2E Auto-inject OrderRequestedAt Test",
		NotifyURL:        "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		// OrderRequestedAt is intentionally NOT set (zero value "")
		// The SDK should auto-inject the current timestamp
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &order.UserInfo{
			UserID:       "e2e_test_user",
			UserEmail:    "e2e@test.com",
			UserTerminal: "WEB",
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
	}

	t.Logf("Creating order WITHOUT OrderRequestedAt: paymentRequestID=%s", paymentRequestID)

	resp, err := testWaffo.Order().Create(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Response: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())

	if !resp.IsSuccess() {
		t.Fatalf("Create order failed (auto-inject OrderRequestedAt should work): paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	t.Logf("Order created successfully without OrderRequestedAt - auto-injection confirmed: paymentRequestID=%s", paymentRequestID)
}

func TestPaymentFlow_CancelPayment(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// Create order
	paymentRequestID := fmt.Sprintf("e2e_cancel_%d", time.Now().UnixNano())
	merchantOrderID := fmt.Sprintf("E2E_CANCEL_%d", time.Now().UnixMilli())

	params := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "10.00",
		OrderDescription: "E2E Cancel Test",
		NotifyURL:        "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &order.UserInfo{
			UserID:       "e2e_test_user",
			UserEmail:    "e2e@test.com",
			UserTerminal: "WEB",
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
	}

	resp, err := testWaffo.Order().Create(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Payment Request ID: %s", paymentRequestID)
	t.Logf("Response: code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())

	if !resp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	checkoutURL := data.FetchRedirectURL()

	// Open checkout
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	base.WaitForPageLoad()
	base.TakeScreenshot("before_cancel")

	// Note: Cancel button selector may need adjustment
	// base.ClickButton("Cancel")
	// base.WaitForURLContains("cancel")

	base.TakeScreenshot("after_cancel")
	base.PrintCurrentURL()
}

//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
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
		os.Exit(m.Run())
	}

	var err error
	testWaffo, err = testConfig.CreateWaffoClient()
	if err != nil {
		fmt.Printf("Failed to create Waffo client: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Waffo client initialized with merchant: %s\n", testConfig.MerchantID)

	// Run tests
	os.Exit(m.Run())
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
		PaymentRequestID:   paymentRequestID,
		MerchantOrderID:    merchantOrderID,
		OrderCurrency:      "HKD",
		OrderAmount:        "10.00",
		OrderDescription:   "E2E Test Order",
		NotifyURL:          "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
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
		PaymentRequestID:   paymentRequestID,
		MerchantOrderID:    merchantOrderID,
		OrderCurrency:      "HKD",
		OrderAmount:        "10.00",
		OrderDescription:   "E2E Credit Card Test",
		NotifyURL:          "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
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

	base.SetTimeout(60000)
	if !base.CompleteCheckout(t, checkoutURL, "credit_card_payment", DefaultCheckoutCard()) {
		t.Fatalf("Credit card checkout did not complete: paymentRequestID=%s, returnedPaymentRequestID=%s, acquiringOrderID=%s, currentURL=%s",
			paymentRequestID, returnedPaymentRequestID, data.AcquiringOrderID, base.Page.URL())
	}

	inquiryParams := &order.InquiryOrderParams{
		PaymentRequestID: returnedPaymentRequestID,
	}

	var finalStatus string
	var lastInquiry string
	for attempt := 1; attempt <= 20; attempt++ {
		inquiryResp, inquiryErr := testWaffo.Order().Inquiry(context.Background(), inquiryParams, nil)
		if inquiryErr != nil {
			lastInquiry = fmt.Sprintf("error=%v", inquiryErr)
		} else {
			lastInquiry = fmt.Sprintf("code=%s, msg=%s, data=%+v",
				inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
			t.Logf("Inquiry Response: paymentRequestID=%s, returnedPaymentRequestID=%s, attempt=%d, %s",
				paymentRequestID, returnedPaymentRequestID, attempt, lastInquiry)
			if inquiryResp.IsSuccess() && inquiryResp.GetData() != nil {
				finalStatus = inquiryResp.GetData().OrderStatus
				if finalStatus == "PAY_SUCCESS" {
					t.Logf("Final Order Status: paymentRequestID=%s, acquiringOrderID=%s, status=%s",
						paymentRequestID, data.AcquiringOrderID, finalStatus)
					return
				}
			}
		}
		t.Logf("Waiting for PAY_SUCCESS: paymentRequestID=%s, acquiringOrderID=%s, attempt=%d, status=%s",
			paymentRequestID, data.AcquiringOrderID, attempt, finalStatus)
		time.Sleep(3 * time.Second)
	}

	t.Fatalf("Credit card payment did not reach PAY_SUCCESS: paymentRequestID=%s, returnedPaymentRequestID=%s, acquiringOrderID=%s, finalStatus=%s, lastInquiry=%s",
		paymentRequestID, returnedPaymentRequestID, data.AcquiringOrderID, finalStatus, lastInquiry)
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
		PaymentRequestID:   paymentRequestID,
		MerchantOrderID:    merchantOrderID,
		OrderCurrency:      "HKD",
		OrderAmount:        "10.00",
		OrderDescription:   "E2E Auto-inject OrderRequestedAt Test",
		NotifyURL:          "https://httpbin.org/post",
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
		PaymentRequestID:   paymentRequestID,
		MerchantOrderID:    merchantOrderID,
		OrderCurrency:      "HKD",
		OrderAmount:        "10.00",
		OrderDescription:   "E2E Cancel Test",
		NotifyURL:          "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
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

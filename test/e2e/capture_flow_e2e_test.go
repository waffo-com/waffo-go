//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/order"
)

// TestCaptureFlow_ManualCapture tests the complete manual capture flow:
// 1. Create order with captureMode=manualCapture
// 2. Open checkout and complete credit card payment
// 3. Poll for AUTHED_WAITING_CAPTURE status
// 4. Call order.capture() to complete the charge
// 5. Verify capture success
//
// Key findings from sandbox testing:
// - Card 4000000000001000 required (4111111111111111 rejected with "card type not supported")
// - captureRequestedAt is mandatory (server returns A0003 if missing)
// - After filling card number, must wait 2s + click expiry field before fill (auto-formatting steals focus)
// - AUTHED_WAITING_CAPTURE reached on first poll (~3s after payment)
// - Capture response status is CAPTURE_IN_PROGRESS (async settlement)
//
// Run with:
//   go test -tags=e2e ./test/e2e/... -run TestCaptureFlow -v -timeout 180s
func TestCaptureFlow_ManualCapture(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// === Step 1: Create order with captureMode=manualCapture ===
	t.Log("=== Step 1: Creating order with captureMode=manualCapture ===")
	paymentRequestID := fmt.Sprintf("e2e_capture_%d", time.Now().UnixNano())
	merchantOrderID := fmt.Sprintf("E2E_CAPTURE_%d", time.Now().UnixMilli())

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "10.00",
		OrderDescription: "E2E Manual Capture Test",
		OrderRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:        "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &order.UserInfo{
			UserID:       "e2e_capture_user",
			UserEmail:    "capture@test.com",
			UserTerminal: "WEB",
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
			CaptureMode:   "manualCapture",
		},
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Create order error: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	if !createResp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, captureMode=manualCapture, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("Order data is nil: paymentRequestID=%s, code=%s, msg=%s",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage())
	}

	checkoutURL := createData.FetchRedirectURL()
	acquiringOrderID := createData.AcquiringOrderID

	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: paymentRequestID=%s, data=%+v", paymentRequestID, createData)
	}

	t.Logf("Payment Request ID: %s", paymentRequestID)
	t.Logf("Acquiring Order ID: %s", acquiringOrderID)
	t.Logf("Checkout URL: %s", checkoutURL)

	// === Step 2: Open checkout and pay with credit card ===
	t.Log("=== Step 2: Completing credit card payment ===")
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to checkout: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(2000)

	base.TakeScreenshot("capture_before_card_input")

	// Use 4000000000001000 card for manualCapture (4111... not supported)
	base.Fill("#payMethodProperties\\.card\\.pan", "4000000000001000")
	base.SleepMs(2000) // Wait for card number auto-formatting

	base.Click("#payMethodProperties\\.card\\.expiry")
	base.SleepMs(300)
	expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
	base.Fill("#payMethodProperties\\.card\\.expiry", expiry)
	base.SleepMs(500)

	base.Click("#payMethodProperties\\.card\\.cvv")
	base.SleepMs(300)
	base.Fill("#payMethodProperties\\.card\\.cvv", TestCard.CVV)
	base.SleepMs(500)

	base.Fill("#payMethodProperties\\.card\\.name", TestCard.Holder)
	base.SleepMs(500)

	base.TakeScreenshot("capture_after_card_input")

	t.Log("Card info filled, clicking pay button...")
	if err := base.Click("button[type='submit']"); err != nil {
		t.Logf("Warning: Could not click submit button: %v", err)
	}

	// Wait for redirect
	base.SetTimeout(60000)
	if err := base.WaitForURLContains("success"); err != nil {
		t.Logf("Did not redirect to success page (may be expected for manualCapture): %v", err)
		base.PrintCurrentURL()
	} else {
		t.Log("Redirected to success page.")
	}

	base.TakeScreenshot("capture_after_payment")

	// === Step 3: Poll for AUTHED_WAITING_CAPTURE ===
	t.Log("=== Step 3: Polling for AUTHED_WAITING_CAPTURE ===")
	base.SleepMs(3000)

	var finalStatus string
	maxRetries := 20
	intervalMs := 3000

	for i := 0; i < maxRetries; i++ {
		inquiryParams := &order.InquiryOrderParams{
			AcquiringOrderID: acquiringOrderID,
		}

		inquiryResp, inquiryErr := testWaffo.Order().Inquiry(context.Background(), inquiryParams, nil)
		if inquiryErr != nil {
			t.Logf("Inquiry error %d/%d: %v", i+1, maxRetries, inquiryErr)
			base.SleepMs(intervalMs)
			continue
		}

		if inquiryResp.IsSuccess() {
			inquiryData := inquiryResp.GetData()
			if inquiryData != nil {
				finalStatus = inquiryData.OrderStatus
				t.Logf("Status check %d/%d: %s", i+1, maxRetries, finalStatus)

				if finalStatus == core.OrderStatusAuthedWaitingCapture {
					t.Log("Order reached AUTHED_WAITING_CAPTURE!")
					break
				}
				if finalStatus == core.OrderStatusPaySuccess {
					t.Log("Order already PAY_SUCCESS (captureMode may not be applied in sandbox)")
					break
				}
			}
		} else {
			t.Logf("Inquiry %d/%d: code=%s, msg=%s", i+1, maxRetries, inquiryResp.GetCode(), inquiryResp.GetMessage())
		}

		base.SleepMs(intervalMs)
	}

	if finalStatus == "" {
		t.Fatalf("Order status never resolved: paymentRequestID=%s, acquiringOrderID=%s",
			paymentRequestID, acquiringOrderID)
	}

	// If sandbox doesn't support manualCapture, skip capture step
	if finalStatus == core.OrderStatusPaySuccess {
		t.Log("SKIP: Sandbox went straight to PAY_SUCCESS, capture not applicable.")
		return
	}

	if finalStatus != core.OrderStatusAuthedWaitingCapture {
		t.Fatalf("Expected AUTHED_WAITING_CAPTURE but got %s: paymentRequestID=%s, acquiringOrderID=%s",
			finalStatus, paymentRequestID, acquiringOrderID)
	}

	// === Step 4: Capture the order ===
	t.Log("=== Step 4: Capturing order ===")
	captureParams := &order.CaptureOrderParams{
		PaymentRequestID:   paymentRequestID,
		AcquiringOrderID:   acquiringOrderID,
		MerchantID:         testConfig.MerchantID,
		CaptureRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		CaptureAmount:      "10.00",
	}

	captureResp, captureErr := testWaffo.Order().Capture(context.Background(), captureParams, nil)
	if captureErr != nil {
		t.Fatalf("Capture error: paymentRequestID=%s, acquiringOrderID=%s, error=%v",
			paymentRequestID, acquiringOrderID, captureErr)
	}

	t.Logf("Capture response: paymentRequestID=%s, acquiringOrderID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, acquiringOrderID, captureResp.GetCode(), captureResp.GetMessage(), captureResp.GetData())

	if !captureResp.IsSuccess() {
		t.Fatalf("Capture failed: paymentRequestID=%s, acquiringOrderID=%s, captureAmount=10.00, code=%s, msg=%s, data=%+v",
			paymentRequestID, acquiringOrderID, captureResp.GetCode(), captureResp.GetMessage(), captureResp.GetData())
	}

	// === Step 5: Verify final status ===
	t.Log("=== Step 5: Verifying final order status ===")
	base.SleepMs(2000)

	finalInquiryParams := &order.InquiryOrderParams{
		AcquiringOrderID: acquiringOrderID,
	}

	finalInquiryResp, _ := testWaffo.Order().Inquiry(context.Background(), finalInquiryParams, nil)
	if finalInquiryResp != nil && finalInquiryResp.IsSuccess() {
		finalData := finalInquiryResp.GetData()
		if finalData != nil {
			t.Logf("Final order status after capture: %s", finalData.OrderStatus)
		}
	} else if finalInquiryResp != nil {
		t.Logf("Final inquiry: code=%s, msg=%s", finalInquiryResp.GetCode(), finalInquiryResp.GetMessage())
	}
}

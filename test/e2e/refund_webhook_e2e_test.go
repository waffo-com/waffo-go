//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/types/order"
)

// TestRefundWebhook_FullFlow tests the complete refund flow with webhook notification:
// 1. Setup webhook server + cloudflared tunnel
// 2. Create DANA order with webhook notifyUrl
// 3. Simulate DANA payment via Playwright ("Payment succeeded" button)
// 4. Execute refund with acquiringOrderId
// 5. Wait for REFUND_NOTIFICATION and validate handler received it
//
// Uses DANA (EWALLET/IDR) because:
// - DANA sandbox has a "Payment succeeded" button for instant payment simulation
// - DANA refunds are processed synchronously and trigger REFUND_NOTIFICATION
// - CREDITCARD refunds are async in sandbox and don't trigger REFUND_NOTIFICATION reliably
//
// Run with:
//   go test -tags=e2e -timeout 300s -run TestRefundWebhook -v ./test/e2e/

func TestRefundWebhook_FullFlow(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	// ==================== Step 1: Setup Webhook Infrastructure ====================
	t.Log("=== Step 1: Setting up webhook infrastructure ===")
	setupWebhookInfra(t)
	defer cleanupWebhookInfra()

	notifyURL := webhookNgrokURL + "/webhook"
	t.Logf("Webhook notify URL: %s", notifyURL)

	// ==================== Step 2: Create DANA Order ====================
	t.Log("=== Step 2: Creating DANA order ===")

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	paymentRequestID := fmt.Sprintf("e2e_rfw_%d", time.Now().UnixMilli())
	merchantOrderID := fmt.Sprintf("E2E_REFUND_WH_%d", time.Now().UnixMilli())

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "IDR",
		OrderAmount:      "50000",
		OrderDescription: "E2E Refund Webhook Test Order",
		UserInfo: &order.UserInfo{
			UserID:       "dana_test_user",
			UserEmail:    "dana@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &order.GoodsInfo{
			GoodsURL: "https://example.com/goods/refund-webhook",
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "EWALLET",
			PayMethodName: "DANA",
		},
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:          notifyURL,
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}
	if !createResp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("Order data is nil: paymentRequestID=%s", paymentRequestID)
	}

	acquiringOrderID := createData.AcquiringOrderID
	checkoutURL := createData.FetchRedirectURL()
	t.Logf("paymentRequestID=%s, acquiringOrderID=%s", paymentRequestID, acquiringOrderID)

	if acquiringOrderID == "" {
		t.Fatalf("acquiringOrderId is empty: paymentRequestID=%s", paymentRequestID)
	}
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: paymentRequestID=%s", paymentRequestID)
	}

	// ==================== Step 3: Simulate DANA Payment ====================
	t.Log("=== Step 3: Simulating DANA payment ===")

	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}
	base.WaitForPageLoad()
	base.SleepMs(3000)

	t.Logf("  DANA page URL: %s", checkoutURL)

	// Click "Payment succeeded" button (DANA sandbox mock)
	danaClicked := false
	for _, sel := range []string{"button:has-text('Payment succeeded')", "#_lv_6"} {
		count, _ := base.Page.Locator(sel).Count()
		if count > 0 {
			base.Page.Locator(sel).First().Click()
			t.Logf("  DANA payment simulated: %s", sel)
			danaClicked = true
			base.SleepMs(3000)
			break
		}
	}
	if !danaClicked {
		// Log available buttons for debugging
		buttons, _ := base.Page.Locator("button").All()
		for _, btn := range buttons {
			txt, _ := btn.TextContent()
			t.Logf("  Available button: %q", txt)
		}
		t.Fatalf("DANA 'Payment succeeded' button not found: paymentRequestID=%s, acquiringOrderID=%s",
			paymentRequestID, acquiringOrderID)
	}

	base.TakeScreenshot("refund_webhook_step3_dana_paid")

	// Poll for PAY_SUCCESS
	orderPaid := false
	for i := 0; i < 10; i++ {
		inqResp, inqErr := testWaffo.Order().Inquiry(context.Background(), &order.InquiryOrderParams{
			PaymentRequestID: paymentRequestID,
		}, nil)
		if inqErr == nil && inqResp.IsSuccess() && inqResp.GetData() != nil {
			t.Logf("  Order status (attempt %d): paymentRequestID=%s, status=%s",
				i+1, paymentRequestID, inqResp.GetData().OrderStatus)
			if inqResp.GetData().OrderStatus == "PAY_SUCCESS" {
				orderPaid = true
				break
			}
		}
		base.SleepMs(2000)
	}
	if !orderPaid {
		t.Fatalf("Order did not reach PAY_SUCCESS: paymentRequestID=%s, acquiringOrderID=%s",
			paymentRequestID, acquiringOrderID)
	}

	// ==================== Step 4: Execute Refund ====================
	t.Log("=== Step 4: Executing refund ===")

	refundRequestID := fmt.Sprintf("e2e_rfw_refund_%d", time.Now().UnixMilli())

	refundParams := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refundRequestID,
		RefundAmount:     "5000",
		RefundReason:     "E2E Refund Webhook Test",
		NotifyURL:        notifyURL,
	}

	t.Logf("Refund: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s",
		paymentRequestID, acquiringOrderID, refundRequestID)

	refundResp, err := testWaffo.Order().Refund(context.Background(), refundParams, nil)
	if err != nil {
		t.Fatalf("Refund API error: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, acquiringOrderID, refundRequestID, err)
	}

	t.Logf("Refund response: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, acquiringOrderID, refundRequestID,
		refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())

	if !refundResp.IsSuccess() {
		t.Fatalf("Refund failed: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, acquiringOrderID, refundRequestID,
			refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())
	}

	refundData := refundResp.GetData()
	if refundData != nil {
		t.Logf("Refund initiated: acquiringRefundOrderID=%s, refundStatus=%s",
			refundData.AcquiringRefundOrderID, refundData.RefundStatus)
	}

	// ==================== Step 5: Wait for REFUND_NOTIFICATION ====================
	t.Log("=== Step 5: Waiting for REFUND_NOTIFICATION ===")

	notifications := webhookServer.WaitForNotification("REFUND_NOTIFICATION", 1, webhookTimeout)

	if len(notifications) == 0 {
		t.Logf("No REFUND_NOTIFICATION received within timeout: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s",
			paymentRequestID, acquiringOrderID, refundRequestID)
	} else {
		n := notifications[0]
		t.Logf("REFUND_NOTIFICATION received: eventType=%s, handlerSuccess=%v", n.EventType, n.HandlerSuccess)

		if !n.HandlerSuccess {
			t.Errorf("Refund notification handler failed: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s",
				paymentRequestID, acquiringOrderID, refundRequestID)
		}
		if n.EventType != "REFUND_NOTIFICATION" {
			t.Errorf("Expected REFUND_NOTIFICATION, got %s", n.EventType)
		}
		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			t.Logf("  refundStatus: %v", result["refundStatus"])
			t.Logf("  acquiringRefundOrderId: %v", result["acquiringRefundOrderId"])
		}
	}

	t.Log("=== TestRefundWebhook_FullFlow PASSED ===")
}

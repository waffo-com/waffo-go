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

	acquiringOrderID := createPaidDANAOrder(
		t,
		base,
		paymentRequestID,
		merchantOrderID,
		"50000",
		"E2E Refund Webhook Test Order",
		notifyURL,
	)

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
	if refundData == nil {
		t.Fatalf("Refund data is nil: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s",
			paymentRequestID, acquiringOrderID, refundRequestID)
	}
	acquiringRefundOrderID := refundData.AcquiringRefundOrderID
	t.Logf("Refund initiated: acquiringRefundOrderID=%s, refundStatus=%s",
		acquiringRefundOrderID, refundData.RefundStatus)

	// ==================== Step 5: Wait for REFUND_NOTIFICATION ====================
	t.Log("=== Step 5: Waiting for REFUND_NOTIFICATION ===")

	n, ok := webhookServer.WaitForNotificationMatching("REFUND_NOTIFICATION", webhookTimeout, func(n ReceivedNotification) bool {
		result, ok := n.Parsed["result"].(map[string]interface{})
		if !ok {
			return false
		}
		return result["refundRequestId"] == refundRequestID ||
			result["acquiringRefundOrderId"] == acquiringRefundOrderID
	})
	if !ok {
		t.Fatalf("No matching REFUND_NOTIFICATION received within timeout: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s",
			paymentRequestID, acquiringOrderID, refundRequestID)
	}
	t.Logf("REFUND_NOTIFICATION received: eventType=%s, handlerSuccess=%v", n.EventType, n.HandlerSuccess)

	if !n.HandlerSuccess {
		t.Errorf("Refund notification handler failed: paymentRequestID=%s, acquiringOrderID=%s, refundRequestID=%s",
			paymentRequestID, acquiringOrderID, refundRequestID)
	}
	if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
		t.Logf("  refundStatus: %v", result["refundStatus"])
		t.Logf("  acquiringRefundOrderId: %v", result["acquiringRefundOrderId"])
	}

	t.Log("=== TestRefundWebhook_FullFlow PASSED ===")
}

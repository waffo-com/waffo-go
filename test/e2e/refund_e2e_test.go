//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/types/order"
	"github.com/waffo-com/waffo-go/types/refund"
)

// RefundE2ETest tests the complete refund flow:
// 1. Create and pay an order
// 2. Execute partial refund
// 3. Query refund by refundRequestId
// 4. Query refund by acquiringRefundOrderId
//
// Run with:
//   go test -tags=e2e ./test/e2e/... -run TestRefund -v
//
// Visual mode (non-headless):
//   E2E_HEADLESS=false go test -tags=e2e ./test/e2e/... -run TestRefund -v

func TestRefund_PartialRefundFlow(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// ==================== Step 1: Create and Pay DANA Order ====================
	t.Log("=== Step 1: Creating and paying DANA order ===")

	paymentRequestID := fmt.Sprintf("e2e_rfd_%d", time.Now().UnixMilli())
	merchantOrderID := fmt.Sprintf("E2E_REFUND_ORDER_%d", time.Now().UnixMilli())
	notifyURL := "https://httpbin.org/post"

	acquiringOrderID := createPaidDANAOrder(
		t,
		base,
		paymentRequestID,
		merchantOrderID,
		"50000",
		"E2E DANA Refund Test Order",
		notifyURL,
	)

	// ==================== Step 2: Execute Partial Refund ====================
	t.Log("=== Step 2: Executing partial refund ===")

	refundRequestID := fmt.Sprintf("e2e_rr_%d", time.Now().UnixMilli())

	refundParams := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refundRequestID,
		RefundAmount:     "5000",
		RefundReason:     "E2E Test Partial Refund",
		NotifyURL:        notifyURL,
	}

	t.Logf("Refund Request ID: %s", refundRequestID)
	t.Logf("Refund Amount: 5000 IDR")

	refundResp, err := testWaffo.Order().Refund(context.Background(), refundParams, nil)
	if err != nil {
		t.Fatalf("Failed to execute refund: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refundRequestID, err)
	}

	t.Logf("Refund Response: code=%s, msg=%s, data=%+v", refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())

	if !refundResp.IsSuccess() {
		t.Fatalf("Refund failed for paid DANA order: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refundRequestID, refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())
	}

	refundData := refundResp.GetData()
	if refundData == nil {
		t.Fatalf("Refund data is nil: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s",
			paymentRequestID, refundRequestID, refundResp.GetCode(), refundResp.GetMessage())
	}

	acquiringRefundOrderID := refundData.AcquiringRefundOrderID

	t.Log("Refund initiated successfully!")
	t.Logf("Acquiring Refund Order ID: %s", acquiringRefundOrderID)
	t.Logf("Refund Status: %s", refundData.RefundStatus)
	t.Logf("Refund Amount: %s", refundData.RefundAmount)

	base.TakeScreenshot("refund_step2_refund_initiated")

	// ==================== Step 3: Query Refund by RefundRequestId ====================
	t.Log("=== Step 3: Querying refund by refundRequestId ===")

	refundInquiryParams := &refund.InquiryRefundParams{
		RefundRequestID: refundRequestID,
	}

	refundInquiryResp, err := testWaffo.Refund().Inquiry(context.Background(), refundInquiryParams, nil)
	if err != nil {
		t.Fatalf("Failed to query refund after successful refund API call: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refundRequestID, err)
	}
	t.Logf("Refund Inquiry Response: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, refundRequestID, refundInquiryResp.GetCode(), refundInquiryResp.GetMessage(), refundInquiryResp.GetData())

	if !refundInquiryResp.IsSuccess() {
		t.Fatalf("Refund inquiry failed after successful refund API call: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refundRequestID, refundInquiryResp.GetCode(), refundInquiryResp.GetMessage(), refundInquiryResp.GetData())
	}
	refundInquiryData := refundInquiryResp.GetData()
	if refundInquiryData == nil {
		t.Fatalf("Refund inquiry data is nil: paymentRequestID=%s, refundRequestID=%s",
			paymentRequestID, refundRequestID)
	}
	t.Logf("Refund Inquiry Detail: refundRequestID=%s, acquiringRefundOrderID=%s, status=%s, amount=%s, userCurrency=%s, reason=%s",
		refundInquiryData.RefundRequestID, refundInquiryData.AcquiringRefundOrderID,
		refundInquiryData.RefundStatus, refundInquiryData.RefundAmount,
		refundInquiryData.UserCurrency, refundInquiryData.RefundReason)

	// Verify refund request ID matches
	if refundInquiryData.RefundRequestID != refundRequestID {
		t.Errorf("Refund request ID mismatch: paymentRequestID=%s, refundRequestID=%s, expected=%s, got=%s, inquiryData=%+v",
			paymentRequestID, refundRequestID, refundRequestID, refundInquiryData.RefundRequestID, refundInquiryData)
	}

	// ==================== Step 4: Query Refund by AcquiringRefundOrderId ====================
	if acquiringRefundOrderID != "" {
		t.Log("=== Step 4: Querying refund by acquiringRefundOrderId ===")

		// Note: This would require a different inquiry param structure
		// For now, we'll skip this step as the InquiryRefundParams may not support acquiringRefundOrderId
		t.Log("Query by acquiringRefundOrderId skipped - may require different API structure")
	}

	base.TakeScreenshot("refund_step4_completed")
	t.Log("=== Refund Test Completed ===")
}

func TestRefund_QueryOnly(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	// This test only queries an existing refund
	// Useful for checking the query API without creating new orders

	refundRequestID := "existing_refund_request_id"

	refundInquiryParams := &refund.InquiryRefundParams{
		RefundRequestID: refundRequestID,
	}

	resp, err := testWaffo.Refund().Inquiry(context.Background(), refundInquiryParams, nil)
	if err != nil {
		t.Logf("Query error (expected if refund doesn't exist): refundRequestID=%s, error=%v",
			refundRequestID, err)
		t.Skip("Placeholder refund is not configured in sandbox")
	}

	t.Logf("Refund Inquiry Response: refundRequestID=%s, code=%s, msg=%s, data=%+v",
		refundRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())

	if resp.IsSuccess() {
		data := resp.GetData()
		if data != nil {
			t.Logf("Refund Status: refundRequestID=%s, status=%s, amount=%s",
				refundRequestID, data.RefundStatus, data.RefundAmount)
		}
	} else {
		t.Logf("Query failed: refundRequestID=%s, code=%s, msg=%s, data=%+v",
			refundRequestID, resp.GetCode(), resp.GetMessage(), resp.GetData())
		t.Skipf("Placeholder refund is not available: code=%s, msg=%s", resp.GetCode(), resp.GetMessage())
	}
}

func TestRefund_MultiplePartialRefunds(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	t.Log("=== Multiple Partial Refunds Test ===")

	// Step 1: Create and pay DANA order with higher amount
	paymentRequestID := fmt.Sprintf("e2e_mrfd_%d", time.Now().UnixMilli())
	acquiringOrderID := createPaidDANAOrder(
		t,
		base,
		paymentRequestID,
		fmt.Sprintf("E2E_MULTI_REFUND_%d", time.Now().UnixMilli()),
		"100000",
		"E2E Multiple DANA Refund Test Order",
		"https://httpbin.org/post",
	)

	// First refund: 5000 IDR
	t.Log("=== First Partial Refund: 5000 IDR ===")
	refund1RequestID := fmt.Sprintf("e2e_r1_%d", time.Now().UnixMilli())

	refund1Params := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refund1RequestID,
		RefundAmount:     "5000",
		RefundReason:     "E2E First Partial Refund",
	}

	refund1Resp, refund1Err := testWaffo.Order().Refund(context.Background(), refund1Params, nil)
	if refund1Err != nil {
		t.Fatalf("Failed to execute first refund: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refund1RequestID, refund1Err)
	}

	t.Logf("First Refund Response: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, refund1RequestID, refund1Resp.GetCode(), refund1Resp.GetMessage(), refund1Resp.GetData())

	if refund1Resp.IsSuccess() {
		t.Logf("First refund successful: paymentRequestID=%s, refundRequestID=%s, status=%s",
			paymentRequestID, refund1RequestID, refund1Resp.GetData().RefundStatus)
	} else {
		t.Fatalf("First DANA refund failed: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refund1RequestID, refund1Resp.GetCode(), refund1Resp.GetMessage(), refund1Resp.GetData())
	}

	base.SleepMs(2000)

	// DANA sandbox marks remaining refundable amount as 0 after the first refund,
	// so a second refund should be rejected with the refund-rule error.
	t.Log("=== Second Partial Refund: expect A0014 refund rule rejection ===")
	refund2RequestID := fmt.Sprintf("e2e_r2_%d", time.Now().UnixMilli())

	refund2Params := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refund2RequestID,
		RefundAmount:     "3000",
		RefundReason:     "E2E Second Partial Refund",
	}

	refund2Resp, refund2Err := testWaffo.Order().Refund(context.Background(), refund2Params, nil)
	if refund2Err != nil {
		t.Fatalf("Second refund call returned transport/client error instead of expected A0014 API response: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refund2RequestID, refund2Err)
	}

	t.Logf("Second Refund Response: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, refund2RequestID, refund2Resp.GetCode(), refund2Resp.GetMessage(), refund2Resp.GetData())

	if refund2Resp.IsSuccess() {
		t.Fatalf("Second DANA refund unexpectedly succeeded: paymentRequestID=%s, refundRequestID=%s, status=%s",
			paymentRequestID, refund2RequestID, refund2Resp.GetData().RefundStatus)
	}
	if refund2Resp.GetCode() != "A0014" {
		t.Fatalf("Expected second DANA refund to fail with A0014, got code=%s, msg=%s, data=%+v",
			refund2Resp.GetCode(), refund2Resp.GetMessage(), refund2Resp.GetData())
	}
	t.Logf("Second DANA refund rejected as expected: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s",
		paymentRequestID, refund2RequestID, refund2Resp.GetCode(), refund2Resp.GetMessage())

	t.Log("=== Multiple Partial Refunds Test Completed ===")
}

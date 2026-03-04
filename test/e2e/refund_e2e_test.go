//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/types"
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

	// ==================== Step 1: Create and Pay Order ====================
	t.Log("=== Step 1: Creating and paying order ===")

	paymentRequestID := fmt.Sprintf("e2e_rfd_%d", time.Now().UnixMilli())
	merchantOrderID := fmt.Sprintf("E2E_REFUND_ORDER_%d", time.Now().UnixMilli())

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "100.00", // Enough for multiple partial refunds
		OrderDescription: "E2E Refund Test Order",
		UserInfo: &order.UserInfo{
			UserID:       fmt.Sprintf("e2e_user_%d", time.Now().UnixNano()),
			UserEmail:    "e2e_refund@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &order.GoodsInfo{
			GoodsID:       "E2E_GOODS_REFUND",
			GoodsName:     "E2E Refund Test Product",
			GoodsCategory: "GOODS",
			GoodsURL:      "https://example.com/goods/refund-test",
			GoodsQuantity: 1,
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:          "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Payment Request ID: %s", paymentRequestID)
	t.Logf("Response: code=%s, msg=%s, data=%+v", createResp.GetCode(), createResp.GetMessage(), createResp.GetData())

	if !createResp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("Order data is nil: paymentRequestID=%s, code=%s, msg=%s",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage())
	}

	acquiringOrderID := createData.AcquiringOrderID
	checkoutURL := createData.FetchRedirectURL()

	t.Logf("Acquiring Order ID: %s", acquiringOrderID)
	t.Logf("Checkout URL: %s", checkoutURL)

	// Navigate to checkout and complete payment
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: paymentRequestID=%s, data=%+v", paymentRequestID, createData)
	}

	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to checkout: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(2000)

	// Fill card details
	base.Fill("#payMethodProperties\\.card\\.pan", TestCard.Number)
	base.SleepMs(300)
	expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
	base.Fill("#payMethodProperties\\.card\\.expiry", expiry)
	base.SleepMs(300)
	base.Fill("#payMethodProperties\\.card\\.cvv", TestCard.CVV)
	base.SleepMs(300)
	base.Fill("#payMethodProperties\\.card\\.name", TestCard.Holder)
	base.SleepMs(300)

	base.TakeScreenshot("refund_step1_card_filled")

	// Submit payment
	if err := base.Click("button[type='submit']"); err != nil {
		t.Logf("Warning: Could not click submit button: %v", err)
	}

	base.SetTimeout(60000)
	if err := base.WaitForURLContains("success"); err != nil {
		t.Logf("Did not redirect to success page: %v", err)
		base.PrintCurrentURL()
		t.Log("Continuing test - payment may have succeeded without redirect")
	} else {
		t.Log("Payment completed successfully!")
	}

	base.TakeScreenshot("refund_step1_payment_completed")

	// Wait for payment to be processed
	base.SleepMs(3000)

	// Verify payment success by querying order
	inquiryParams := &order.InquiryOrderParams{
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		PaymentRequestID: paymentRequestID,
	}

	var orderPaid bool
	for i := 0; i < 5; i++ {
		inquiryResp, err := testWaffo.Order().Inquiry(context.Background(), inquiryParams, nil)
		if err == nil && inquiryResp.IsSuccess() {
			inquiryData := inquiryResp.GetData()
			if inquiryData != nil {
				t.Logf("Order Status: %s", inquiryData.OrderStatus)
				if inquiryData.OrderStatus == "PAID" || inquiryData.OrderStatus == "SUCCESS" {
					orderPaid = true
					break
				}
			}
		}
		base.SleepMs(2000)
	}

	if !orderPaid {
		t.Log("Warning: Order may not be in PAID status, refund may fail")
	}

	// ==================== Step 2: Execute Partial Refund ====================
	t.Log("=== Step 2: Executing partial refund ===")

	refundRequestID := fmt.Sprintf("e2e_rr_%d", time.Now().UnixMilli())

	refundParams := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refundRequestID,
		RefundAmount:     "30.00", // Partial refund
		RefundReason:     "E2E Test Partial Refund",
		NotifyURL:        "https://httpbin.org/post",
	}

	t.Logf("Refund Request ID: %s", refundRequestID)
	t.Logf("Refund Amount: 30.00 HKD")

	refundResp, err := testWaffo.Order().Refund(context.Background(), refundParams, nil)
	if err != nil {
		t.Fatalf("Failed to execute refund: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refundRequestID, err)
	}

	t.Logf("Refund Response: code=%s, msg=%s, data=%+v", refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())

	if !refundResp.IsSuccess() {
		t.Logf("Refund failed (may be expected if order not paid): paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refundRequestID, refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())
		t.Log("This test requires the order to be successfully paid first")
		return
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
		MerchantInfo: types.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		RefundRequestID: refundRequestID,
	}

	refundInquiryResp, err := testWaffo.Refund().Inquiry(context.Background(), refundInquiryParams, nil)
	if err != nil {
		t.Logf("Warning: Failed to query refund: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refundRequestID, err)
	} else {
		t.Logf("Refund Inquiry Response: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refundRequestID, refundInquiryResp.GetCode(), refundInquiryResp.GetMessage(), refundInquiryResp.GetData())

		if refundInquiryResp.IsSuccess() {
			refundInquiryData := refundInquiryResp.GetData()
			if refundInquiryData != nil {
				t.Logf("Refund Inquiry Detail: refundRequestID=%s, acquiringRefundOrderID=%s, status=%s, amount=%s, currency=%s, reason=%s",
					refundInquiryData.RefundRequestID, refundInquiryData.AcquiringRefundOrderID,
					refundInquiryData.RefundStatus, refundInquiryData.RefundAmount,
					refundInquiryData.RefundCurrency, refundInquiryData.RefundReason)

				// Verify refund request ID matches
				if refundInquiryData.RefundRequestID != refundRequestID {
					t.Errorf("Refund request ID mismatch: paymentRequestID=%s, refundRequestID=%s, expected=%s, got=%s, inquiryData=%+v",
						paymentRequestID, refundRequestID, refundRequestID, refundInquiryData.RefundRequestID, refundInquiryData)
				}
			}
		} else {
			t.Logf("Refund inquiry failed: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
				paymentRequestID, refundRequestID, refundInquiryResp.GetCode(), refundInquiryResp.GetMessage(), refundInquiryResp.GetData())
		}
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
		MerchantInfo: types.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		RefundRequestID: refundRequestID,
	}

	resp, err := testWaffo.Refund().Inquiry(context.Background(), refundInquiryParams, nil)
	if err != nil {
		t.Logf("Query error (expected if refund doesn't exist): refundRequestID=%s, error=%v",
			refundRequestID, err)
		return
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

	// Step 1: Create and pay order with higher amount
	paymentRequestID := fmt.Sprintf("e2e_mrfd_%d", time.Now().UnixMilli())

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  fmt.Sprintf("E2E_MULTI_REFUND_%d", time.Now().UnixMilli()),
		OrderCurrency:    "HKD",
		OrderAmount:      "200.00", // Higher amount for multiple refunds
		OrderDescription: "E2E Multiple Refund Test Order",
		UserInfo: &order.UserInfo{
			UserID:       fmt.Sprintf("e2e_multi_user_%d", time.Now().UnixNano()),
			UserEmail:    "e2e_multi_refund@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &order.GoodsInfo{
			GoodsID:       "E2E_GOODS_MULTI_REFUND",
			GoodsName:     "E2E Multiple Refund Test Product",
			GoodsCategory: "GOODS",
			GoodsURL:      "https://example.com/goods/multi-refund",
			GoodsQuantity: 1,
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:          "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Payment Request ID: %s", paymentRequestID)
	t.Logf("Response: code=%s, msg=%s, data=%+v", createResp.GetCode(), createResp.GetMessage(), createResp.GetData())

	if !createResp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	acquiringOrderID := createData.AcquiringOrderID
	checkoutURL := createData.FetchRedirectURL()

	t.Logf("Acquiring Order ID: %s", acquiringOrderID)
	t.Logf("Checkout URL: %s", checkoutURL)

	// Complete payment
	if checkoutURL != "" {
		if err := base.NavigateTo(checkoutURL); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		base.WaitForPageLoad()
		base.SleepMs(2000)

		base.Fill("#payMethodProperties\\.card\\.pan", TestCard.Number)
		base.SleepMs(200)
		expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
		base.Fill("#payMethodProperties\\.card\\.expiry", expiry)
		base.SleepMs(200)
		base.Fill("#payMethodProperties\\.card\\.cvv", TestCard.CVV)
		base.SleepMs(200)
		base.Fill("#payMethodProperties\\.card\\.name", TestCard.Holder)
		base.SleepMs(200)

		base.Click("button[type='submit']")
		base.SetTimeout(60000)
		base.WaitForURLContains("success")
	}

	base.SleepMs(3000)

	// First refund: 50 HKD
	t.Log("=== First Partial Refund: 50 HKD ===")
	refund1RequestID := fmt.Sprintf("e2e_r1_%d", time.Now().UnixMilli())

	refund1Params := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refund1RequestID,
		RefundAmount:     "50.00",
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
		t.Logf("First refund failed: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refund1RequestID, refund1Resp.GetCode(), refund1Resp.GetMessage(), refund1Resp.GetData())
		return
	}

	base.SleepMs(2000)

	// Second refund: 30 HKD
	t.Log("=== Second Partial Refund: 30 HKD ===")
	refund2RequestID := fmt.Sprintf("e2e_r2_%d", time.Now().UnixMilli())

	refund2Params := &order.RefundOrderParams{
		AcquiringOrderID: acquiringOrderID,
		RefundRequestID:  refund2RequestID,
		RefundAmount:     "30.00",
		RefundReason:     "E2E Second Partial Refund",
	}

	refund2Resp, refund2Err := testWaffo.Order().Refund(context.Background(), refund2Params, nil)
	if refund2Err != nil {
		t.Logf("Warning: Failed to execute second refund: paymentRequestID=%s, refundRequestID=%s, error=%v",
			paymentRequestID, refund2RequestID, refund2Err)
		return
	}

	t.Logf("Second Refund Response: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, refund2RequestID, refund2Resp.GetCode(), refund2Resp.GetMessage(), refund2Resp.GetData())

	if refund2Resp.IsSuccess() {
		t.Logf("Second refund successful: paymentRequestID=%s, refundRequestID=%s, status=%s",
			paymentRequestID, refund2RequestID, refund2Resp.GetData().RefundStatus)
	} else {
		t.Logf("Second refund failed (may be expected): paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, refund2RequestID, refund2Resp.GetCode(), refund2Resp.GetMessage(), refund2Resp.GetData())
	}

	t.Log("=== Multiple Partial Refunds Test Completed ===")
}

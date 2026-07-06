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

func TestPaymentFailure_DANAOrderClose(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	paymentRequestID := fmt.Sprintf("e2e_fail_%d", time.Now().UnixMilli())
	merchantOrderID := fmt.Sprintf("E2E_FAIL_%d", time.Now().UnixMilli())

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "IDR",
		OrderAmount:      "10000",
		OrderDescription: "E2E DANA Payment Failure Test",
		UserInfo: &order.UserInfo{
			UserID:       fmt.Sprintf("dana_fail_user_%d", time.Now().UnixNano()),
			UserEmail:    "dana-fail@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &order.GoodsInfo{
			GoodsID:       "DANA_FAIL_GOODS",
			GoodsName:     "DANA Failure Test Product",
			GoodsCategory: "GOODS",
			GoodsURL:      "https://example.com/goods/dana-failure",
			GoodsQuantity: 1,
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
		NotifyURL:          "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create DANA failure order: paymentRequestID=%s, error=%v",
			paymentRequestID, err)
	}
	if !createResp.IsSuccess() {
		t.Fatalf("Create DANA failure order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("DANA failure order data is nil: paymentRequestID=%s", paymentRequestID)
	}

	acquiringOrderID := createData.AcquiringOrderID
	checkoutURL := createData.FetchRedirectURL()
	if acquiringOrderID == "" {
		t.Fatalf("DANA failure acquiringOrderId is empty: paymentRequestID=%s", paymentRequestID)
	}
	if checkoutURL == "" {
		t.Fatalf("DANA failure checkout URL is empty: paymentRequestID=%s, data=%+v",
			paymentRequestID, createData)
	}

	simulateDANAPaymentFailure(t, base, checkoutURL, paymentRequestID, acquiringOrderID)
	waitForDANAOrderStatus(t, paymentRequestID, acquiringOrderID, "ORDER_CLOSE")
}

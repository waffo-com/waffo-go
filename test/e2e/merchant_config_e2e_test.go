//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/types/merchant"
	"github.com/waffo-com/waffo-go/types/order"
)

// TestMerchantConfig_InquiryMerchantConfig tests the merchant config inquiry API
func TestMerchantConfig_InquiryMerchantConfig(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	params := &merchant.InquiryMerchantConfigParams{
		MerchantID: testConfig.MerchantID,
	}

	resp, err := testWaffo.MerchantConfig().Inquiry(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Merchant config inquiry error: merchantID=%s, error=%v",
			testConfig.MerchantID, err)
	}

	if !resp.IsSuccess() {
		t.Fatalf("Merchant config inquiry failed: merchantID=%s, code=%s, msg=%s, data=%+v",
			testConfig.MerchantID, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	if data == nil {
		t.Fatalf("Merchant config data is nil: merchantID=%s, code=%s, msg=%s",
			testConfig.MerchantID, resp.GetCode(), resp.GetMessage())
	}

	t.Logf("Merchant Config: merchantID=%s, totalDailyLimit=%v, transactionLimit=%v",
		data.MerchantID, data.TotalDailyLimit, data.TransactionLimit)
}

// TestMerchantConfig_InquiryPayMethodConfig tests the pay method config inquiry API
func TestMerchantConfig_InquiryPayMethodConfig(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	params := &merchant.InquiryPayMethodConfigParams{
		MerchantID: testConfig.MerchantID,
	}

	resp, err := testWaffo.PayMethodConfig().Inquiry(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Pay method config inquiry error: merchantID=%s, error=%v",
			testConfig.MerchantID, err)
	}

	if !resp.IsSuccess() {
		t.Fatalf("Pay method config inquiry failed: merchantID=%s, code=%s, msg=%s, data=%+v",
			testConfig.MerchantID, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	if data == nil {
		t.Fatalf("Pay method config data is nil: merchantID=%s, code=%s, msg=%s",
			testConfig.MerchantID, resp.GetCode(), resp.GetMessage())
	}

	t.Logf("Pay Method Config: merchantID=%s, payMethodDetails=%+v", testConfig.MerchantID, data.PayMethodDetails)
}

// TestOrderFlow_CancelOrder tests creating an order and then canceling it
func TestOrderFlow_CancelOrder(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	// Step 1: Create order
	paymentRequestID := fmt.Sprintf("e2e_cancel_%d", time.Now().UnixNano())
	merchantOrderID := fmt.Sprintf("E2E_CANCEL_%d", time.Now().UnixMilli())

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "HKD",
		OrderAmount:      "10.00",
		OrderDescription: "E2E Cancel Test Order",
		OrderRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:        "https://httpbin.org/post",
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
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

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Create order error: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	if !createResp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	t.Logf("Order created: paymentRequestID=%s, code=%s, msg=%s",
		paymentRequestID, createResp.GetCode(), createResp.GetMessage())

	// Step 2: Cancel order
	cancelParams := &order.CancelOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantID:       testConfig.MerchantID,
		OrderRequestedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	cancelResp, err := testWaffo.Order().Cancel(context.Background(), cancelParams, nil)
	if err != nil {
		t.Fatalf("Cancel order error: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}

	t.Logf("Cancel order response: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
		paymentRequestID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())

	if cancelResp.IsSuccess() {
		data := cancelResp.GetData()
		if data != nil {
			t.Logf("Order cancelled: paymentRequestID=%s, orderStatus=%s",
				data.PaymentRequestID, data.OrderStatus)
		}
	} else {
		t.Logf("Cancel response (may be expected for unpaid orders): paymentRequestID=%s, code=%s, msg=%s",
			paymentRequestID, cancelResp.GetCode(), cancelResp.GetMessage())
	}
}


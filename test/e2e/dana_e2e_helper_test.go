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

func createPaidDANAOrder(t *testing.T, base *BaseE2ETest, paymentRequestID, merchantOrderID, amount, description, notifyURL string) string {
	t.Helper()

	createParams := &order.CreateOrderParams{
		PaymentRequestID: paymentRequestID,
		MerchantOrderID:  merchantOrderID,
		OrderCurrency:    "IDR",
		OrderAmount:      amount,
		OrderDescription: description,
		UserInfo: &order.UserInfo{
			UserID:       fmt.Sprintf("dana_user_%d", time.Now().UnixNano()),
			UserEmail:    "dana@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &order.GoodsInfo{
			GoodsID:       "DANA_REFUND_GOODS",
			GoodsName:     "DANA Refund Test Product",
			GoodsCategory: "GOODS",
			GoodsURL:      "https://example.com/goods/dana-refund",
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
		NotifyURL:          notifyURL,
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create DANA order: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}
	if !createResp.IsSuccess() {
		t.Fatalf("Create DANA order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			paymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("DANA order data is nil: paymentRequestID=%s", paymentRequestID)
	}

	acquiringOrderID := createData.AcquiringOrderID
	checkoutURL := createData.FetchRedirectURL()
	t.Logf("DANA order created: paymentRequestID=%s, acquiringOrderID=%s", paymentRequestID, acquiringOrderID)
	t.Logf("DANA checkout URL: %s", checkoutURL)

	if acquiringOrderID == "" {
		t.Fatalf("DANA acquiringOrderId is empty: paymentRequestID=%s", paymentRequestID)
	}
	if checkoutURL == "" {
		t.Fatalf("DANA checkout URL is empty: paymentRequestID=%s, data=%+v", paymentRequestID, createData)
	}

	simulateDANAPayment(t, base, checkoutURL, paymentRequestID, acquiringOrderID)
	waitForDANAOrderPaid(t, paymentRequestID, acquiringOrderID)
	return acquiringOrderID
}

func simulateDANAPayment(t *testing.T, base *BaseE2ETest, checkoutURL, paymentRequestID, acquiringOrderID string) {
	t.Helper()

	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to DANA checkout: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}
	base.WaitForPageLoad()
	base.SleepMs(3000)

	for _, sel := range []string{
		"button:has-text('Payment succeeded')",
		"button:has-text('付款成功')",
		"#_lv_6",
	} {
		count, _ := base.Page.Locator(sel).Count()
		if count == 0 {
			continue
		}
		if err := base.Page.Locator(sel).First().Click(); err != nil {
			t.Fatalf("Failed to click DANA success button %s: paymentRequestID=%s, error=%v",
				sel, paymentRequestID, err)
		}
		t.Logf("DANA payment simulated: selector=%s, paymentRequestID=%s", sel, paymentRequestID)
		base.SleepMs(3000)
		return
	}

	buttons, _ := base.Page.Locator("button").All()
	for _, btn := range buttons {
		txt, _ := btn.TextContent()
		t.Logf("Available DANA button: %q", txt)
	}
	t.Fatalf("DANA 'Payment succeeded' button not found: paymentRequestID=%s, acquiringOrderID=%s",
		paymentRequestID, acquiringOrderID)
}

func waitForDANAOrderPaid(t *testing.T, paymentRequestID, acquiringOrderID string) {
	t.Helper()

	for i := 0; i < 10; i++ {
		inqResp, inqErr := testWaffo.Order().Inquiry(context.Background(), &order.InquiryOrderParams{
			PaymentRequestID: paymentRequestID,
		}, nil)
		if inqErr == nil && inqResp.IsSuccess() && inqResp.GetData() != nil {
			status := inqResp.GetData().OrderStatus
			t.Logf("DANA order status (attempt %d): paymentRequestID=%s, status=%s",
				i+1, paymentRequestID, status)
			if status == "PAY_SUCCESS" {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("DANA order did not reach PAY_SUCCESS: paymentRequestID=%s, acquiringOrderID=%s",
		paymentRequestID, acquiringOrderID)
}

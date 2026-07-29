//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/v2/types/order"
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
		"button:has-text('Payment Succeeded')",
		"button:has-text('Mock Payment Success')",
		"button:has-text('Simulate Success')",
		"button:has-text('Pembayaran berhasil')",
		"button:has-text('berhasil')",
		"button:has-text('付款成功')",
		"button:has-text('支付成功')",
		"button:has-text('模擬付款成功')",
		"button:has-text('模拟付款成功')",
		"a:has-text('Payment succeeded')",
		"a:has-text('Pembayaran berhasil')",
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

	buttons, _ := base.Page.Locator("button, a").All()
	for _, btn := range buttons {
		txt, _ := btn.TextContent()
		t.Logf("Available DANA button: %q", txt)
		normalized := strings.ToLower(strings.TrimSpace(txt))
		if strings.Contains(normalized, "payment succeeded") ||
			strings.Contains(normalized, "mock payment success") ||
			strings.Contains(normalized, "simulate success") ||
			strings.Contains(normalized, "pembayaran berhasil") ||
			strings.Contains(normalized, "berhasil") ||
			strings.Contains(txt, "付款成功") ||
			strings.Contains(txt, "支付成功") ||
			strings.Contains(txt, "模擬付款成功") ||
			strings.Contains(txt, "模拟付款成功") {
			if err := btn.Click(); err != nil {
				t.Fatalf("Failed to click DANA success action by text %q: paymentRequestID=%s, error=%v",
					txt, paymentRequestID, err)
			}
			t.Logf("DANA payment simulated by text: %q, paymentRequestID=%s", txt, paymentRequestID)
			base.SleepMs(3000)
			return
		}
	}
	t.Fatalf("DANA 'Payment succeeded' button not found: paymentRequestID=%s, acquiringOrderID=%s",
		paymentRequestID, acquiringOrderID)
}

func waitForDANAOrderPaid(t *testing.T, paymentRequestID, acquiringOrderID string) {
	t.Helper()

	waitForDANAOrderStatus(t, paymentRequestID, acquiringOrderID, "PAY_SUCCESS")
}

func simulateDANAPaymentFailure(t *testing.T, base *BaseE2ETest, checkoutURL, paymentRequestID, acquiringOrderID string) {
	t.Helper()

	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to DANA checkout: paymentRequestID=%s, error=%v", paymentRequestID, err)
	}
	base.WaitForPageLoad()
	base.SleepMs(3000)

	for _, sel := range []string{
		"button:has-text('Payment failed')",
		"button:has-text('Payment Failed')",
		"button:has-text('Mock Payment Failed')",
		"button:has-text('Simulate Failure')",
		"button:has-text('Fail')",
		"button:has-text('Pembayaran gagal')",
		"button:has-text('gagal')",
		"button:has-text('支付失败')",
		"button:has-text('支付失敗')",
		"button:has-text('付款失败')",
		"button:has-text('付款失敗')",
		"a:has-text('Payment failed')",
		"a:has-text('Pembayaran gagal')",
	} {
		count, _ := base.Page.Locator(sel).Count()
		if count == 0 {
			continue
		}
		if err := base.Page.Locator(sel).First().Click(); err != nil {
			t.Fatalf("Failed to click DANA failure button %s: paymentRequestID=%s, error=%v",
				sel, paymentRequestID, err)
		}
		t.Logf("DANA failure simulated: selector=%s, paymentRequestID=%s", sel, paymentRequestID)
		base.SleepMs(3000)
		return
	}

	buttons, _ := base.Page.Locator("button, a").All()
	for _, btn := range buttons {
		txt, _ := btn.TextContent()
		t.Logf("Available DANA button: %q", txt)
		normalized := strings.ToLower(strings.TrimSpace(txt))
		if strings.Contains(normalized, "payment failed") ||
			strings.Contains(normalized, "mock payment failed") ||
			strings.Contains(normalized, "simulate failure") ||
			strings.Contains(normalized, "pembayaran gagal") ||
			strings.Contains(normalized, "gagal") ||
			strings.Contains(normalized, "fail") ||
			strings.Contains(txt, "支付失败") ||
			strings.Contains(txt, "支付失敗") ||
			strings.Contains(txt, "付款失败") ||
			strings.Contains(txt, "付款失敗") {
			if err := btn.Click(); err != nil {
				t.Fatalf("Failed to click DANA failure action by text %q: paymentRequestID=%s, error=%v",
					txt, paymentRequestID, err)
			}
			t.Logf("DANA failure simulated by text: %q, paymentRequestID=%s", txt, paymentRequestID)
			base.SleepMs(3000)
			return
		}
	}
	t.Fatalf("DANA 'Payment failed' button not found: paymentRequestID=%s, acquiringOrderID=%s",
		paymentRequestID, acquiringOrderID)
}

func waitForDANAOrderStatus(t *testing.T, paymentRequestID, acquiringOrderID, expectedStatus string) {
	t.Helper()

	var lastResponse string
	for i := 0; i < 15; i++ {
		inqResp, inqErr := testWaffo.Order().Inquiry(context.Background(), &order.InquiryOrderParams{
			AcquiringOrderID: acquiringOrderID,
		}, nil)
		if inqErr != nil {
			lastResponse = fmt.Sprintf("error=%v", inqErr)
		} else {
			lastResponse = fmt.Sprintf("code=%s, msg=%s, data=%+v",
				inqResp.GetCode(), inqResp.GetMessage(), inqResp.GetData())
		}
		if inqErr == nil && inqResp.IsSuccess() && inqResp.GetData() != nil {
			status := inqResp.GetData().OrderStatus
			t.Logf("DANA order status (attempt %d): paymentRequestID=%s, acquiringOrderID=%s, status=%s",
				i+1, paymentRequestID, acquiringOrderID, status)
			if status == expectedStatus {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("DANA order did not reach %s: paymentRequestID=%s, acquiringOrderID=%s, lastInquiry=%s",
		expectedStatus, paymentRequestID, acquiringOrderID, lastResponse)
}

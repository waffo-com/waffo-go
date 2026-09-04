// README demo mirror tests that run fully offline.
//
// Unlike readme_demo_test.go (sandbox-backed and explicitly enabled with WAFFO_E2E=1),
// these tests stub the HTTP transport with a locally generated key pair, so
// they always run and prove the README demo call patterns compile and behave
// as documented: custom transport injection, order capture, and subscription
// update.
package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	waffo "github.com/waffo-com/waffo-go/v2"
	"github.com/waffo-com/waffo-go/v2/config"
	"github.com/waffo-com/waffo-go/v2/net"
	"github.com/waffo-com/waffo-go/v2/types/order"
	"github.com/waffo-com/waffo-go/v2/types/refund"
	"github.com/waffo-com/waffo-go/v2/types/subscription"
	"github.com/waffo-com/waffo-go/v2/utils"
)

func TestReadmeDemoOffline_ReadmesUseCurrentAPIsAndSafeRequestIDs(t *testing.T) {
	readmePaths := []string{"../README.md", "../README_CN.md"}
	// Standalone Go module checkouts do not include the monorepo source READMEs.
	if _, err := os.Stat("../../../README.md"); err == nil {
		readmePaths = append(readmePaths, "../../../README.md", "../../../README_CN.md")
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect monorepo README: %v", err)
	}

	for _, readmePath := range readmePaths {
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("read %s: %v", readmePath, err)
		}
		for _, stale := range []string{
			`fmt.Sprintf("%x", time.Now().UnixNano())[:32]`,
			"client.Order().Query(",
			"client.Order().Close(",
			"client.Refund().Query(",
			"order.QueryOrderParams",
			"order.CloseOrderParams",
			"refund.QueryRefundParams",
			"SubscriptionPlanID:",
			"BillingCycle:",
		} {
			if strings.Contains(string(content), stale) {
				t.Errorf("%s contains stale or unsafe README code %q", readmePath, stale)
			}
		}
	}

	requestID := fmt.Sprintf("%032x", time.Now().UnixNano())
	if len(requestID) != 32 {
		t.Fatalf("README request ID format produced %d characters: %q", len(requestID), requestID)
	}
}

func TestReadmeDemoOffline_CoreAPIUsageCompilesAndRuns(t *testing.T) {
	client, transport := newOfflineDemoClient(t, nil)
	ctx := context.Background()

	orderCreate, err := client.Order().Create(ctx, &order.CreateOrderParams{
		PaymentRequestID: "REQ-123",
		MerchantOrderID:  "ORDER-123",
		OrderAmount:      "99.99",
		OrderCurrency:    "USD",
		OrderDescription: "Premium Plan",
		NotifyURL:        "https://example.com/webhook",
		UserInfo:         &order.UserInfo{UserID: "USER-123"},
		PaymentInfo:      &order.PaymentInfo{ProductName: "ONE_TIME_PAYMENT"},
	}, nil)
	if err != nil || !orderCreate.IsSuccess() {
		t.Fatalf("README order create failed: err=%v, response=%+v", err, orderCreate)
	}

	orderInquiry, err := client.Order().Inquiry(ctx, &order.InquiryOrderParams{PaymentRequestID: "REQ-123"}, nil)
	if err != nil || !orderInquiry.IsSuccess() {
		t.Fatalf("README order inquiry failed: err=%v, response=%+v", err, orderInquiry)
	}

	orderCancel, err := client.Order().Cancel(ctx, &order.CancelOrderParams{PaymentRequestID: "REQ-123"}, nil)
	if err != nil || !orderCancel.IsSuccess() {
		t.Fatalf("README order cancel failed: err=%v, response=%+v", err, orderCancel)
	}

	subscriptionCreate, err := client.Subscription().Create(ctx, &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    "SUB-REQ-123",
		MerchantSubscriptionID: "MERCHANT-SUB-123",
		Currency:               "USD",
		Amount:                 "9.99",
		ProductInfo: &subscription.ProductInfo{
			Description:    "Premium Plan",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
		},
		UserInfo:    &subscription.SubscriptionUserInfo{UserID: "USER-123"},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{ProductName: "SUBSCRIPTION"},
		NotifyURL:   "https://example.com/webhook",
	}, nil)
	if err != nil || !subscriptionCreate.IsSuccess() {
		t.Fatalf("README subscription create failed: err=%v, response=%+v", err, subscriptionCreate)
	}

	subscriptionInquiry, err := client.Subscription().Inquiry(ctx, &subscription.InquirySubscriptionParams{
		SubscriptionRequest: "SUB-REQ-123",
	}, nil)
	if err != nil || !subscriptionInquiry.IsSuccess() {
		t.Fatalf("README subscription inquiry failed: err=%v, response=%+v", err, subscriptionInquiry)
	}

	subscriptionCancel, err := client.Subscription().Cancel(ctx, &subscription.CancelSubscriptionParams{
		SubscriptionID: "SUB-123",
	}, nil)
	if err != nil || !subscriptionCancel.IsSuccess() {
		t.Fatalf("README subscription cancel failed: err=%v, response=%+v", err, subscriptionCancel)
	}

	refundInquiry, err := client.Refund().Inquiry(ctx, &refund.InquiryRefundParams{RefundRequestID: "REFUND-123"}, nil)
	if err != nil || !refundInquiry.IsSuccess() {
		t.Fatalf("README refund inquiry failed: err=%v, response=%+v", err, refundInquiry)
	}

	if len(transport.requests) != 7 {
		t.Fatalf("expected 7 README API requests, got %d", len(transport.requests))
	}
}

// recordingTransport mirrors the README "Custom HTTP Transport" demo shape:
// a struct implementing net.HttpTransport via Send(ctx, req).
type recordingTransport struct {
	privateKey string
	data       map[string]interface{}
	requests   []*net.HttpRequest
}

func (t *recordingTransport) Send(ctx context.Context, req *net.HttpRequest) (*net.HttpResponse, error) {
	t.requests = append(t.requests, req)
	payload := map[string]interface{}{"code": "0", "msg": "success"}
	if t.data != nil {
		payload["data"] = t.data
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	signature, err := utils.Sign(string(body), t.privateKey)
	if err != nil {
		return nil, err
	}
	return &net.HttpResponse{
		StatusCode: 200,
		Headers:    map[string]string{"X-SIGNATURE": signature},
		Body:       body,
	}, nil
}

func (t *recordingTransport) sentBody(tb testing.TB) map[string]interface{} {
	tb.Helper()
	if len(t.requests) == 0 {
		tb.Fatal("no request was sent through the custom transport")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(t.requests[len(t.requests)-1].Body, &body); err != nil {
		tb.Fatalf("request body is not valid JSON: %v", err)
	}
	return body
}

func newOfflineDemoClient(t *testing.T, data map[string]interface{}) (*waffo.Waffo, *recordingTransport) {
	t.Helper()
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair failed: %v", err)
	}
	transport := &recordingTransport{privateKey: keyPair.PrivateKey, data: data}

	// Mirrors the README custom transport demo: CustomTransport(...) on the builder.
	cfg, err := config.NewConfigBuilder().
		APIKey("your-api-key").
		PrivateKey(keyPair.PrivateKey).
		WaffoPublicKey(keyPair.PublicKey).
		MerchantID("your-merchant-id").
		Environment(config.Sandbox).
		CustomTransport(transport).
		Build()
	if err != nil {
		t.Fatalf("build config failed: %v", err)
	}
	return waffo.New(cfg), transport
}

func TestReadmeDemoOffline_CustomTransportIsUsedByTheClient(t *testing.T) {
	client, transport := newOfflineDemoClient(t, nil)

	resp, err := client.Order().Inquiry(context.Background(), &order.InquiryOrderParams{
		PaymentRequestID: "req_readme_demo",
	}, nil)
	if err != nil {
		t.Fatalf("inquiry through custom transport failed: %v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("inquiry failed: code=%s, msg=%s", resp.GetCode(), resp.GetMessage())
	}
	if len(transport.requests) != 1 {
		t.Fatalf("expected 1 request through custom transport, got %d", len(transport.requests))
	}
}

func TestReadmeDemoOffline_CaptureOrder(t *testing.T) {
	client, transport := newOfflineDemoClient(t, map[string]interface{}{
		"orderStatus": "PAY_SUCCESS",
	})

	resp, err := client.Order().Capture(context.Background(), &order.CaptureOrderParams{
		AcquiringOrderID: "acquiring_order_id",
		CaptureAmount:    "100.00",
	}, nil)
	if err != nil {
		t.Fatalf("capture failed: acquiringOrderId=acquiring_order_id, err=%v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("capture failed: acquiringOrderId=acquiring_order_id, code=%s, msg=%s, data=%+v",
			resp.GetCode(), resp.GetMessage(), resp.GetData())
	}
	if got := resp.GetData().OrderStatus; got != "PAY_SUCCESS" {
		t.Fatalf("unexpected order status: %s", got)
	}

	body := transport.sentBody(t)
	if body["captureAmount"] != "100.00" {
		t.Fatalf("captureAmount not sent: %v", body["captureAmount"])
	}
	// The README documents CaptureRequestedAt auto-injection when not provided.
	if _, ok := body["captureRequestedAt"]; !ok {
		t.Fatal("captureRequestedAt was not auto-injected")
	}
}

func TestReadmeDemoOffline_UpdateSubscription(t *testing.T) {
	client, transport := newOfflineDemoClient(t, map[string]interface{}{
		"previousAmount":      "99.00",
		"newAmount":           "129.00",
		"nextEffectivePeriod": "2",
		"topupInfo": map[string]interface{}{
			"acquiringOrderId": "A_TOPUP_001",
			"topupStatus":      "AUTHORIZATION_REQUIRED",
		},
	})

	resp, err := client.Subscription().Update(context.Background(), &subscription.UpdateSubscriptionParams{
		SubscriptionID: "subscription_id",
		Amount:         "129.00",
		ProductInfo: &subscription.UpdateProductInfo{
			TrialPeriodAmount: "0",
			ScheduledAmounts: []subscription.ScheduledAmount{
				{Period: "2", Amount: "99.00"},
				{Period: "3", Amount: "129.00"},
			},
		},
		TopupInfo: &subscription.SubscriptionUpdateTopupInfo{
			TopupRequest:   "mimo-topup-001",
			TopupAmount:    "30.00",
			Description:    "Seat add-on 3 seats",
			OrderExpiredAt: "2030-01-01T00:00:00Z",
		},
	}, nil)
	if err != nil {
		t.Fatalf("update failed: subscriptionId=subscription_id, err=%v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("update failed: subscriptionId=subscription_id, code=%s, msg=%s, data=%+v",
			resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	if data.PreviousAmount != "99.00" || data.NewAmount != "129.00" || data.NextEffectivePeriod != "2" {
		t.Fatalf("unexpected update data: %+v", data)
	}
	if data.TopupInfo == nil || data.TopupInfo.AcquiringOrderID != "A_TOPUP_001" ||
		data.TopupInfo.TopupStatus != "AUTHORIZATION_REQUIRED" {
		t.Fatalf("unexpected topup data: %+v", data.TopupInfo)
	}

	body := transport.sentBody(t)
	productInfo, ok := body["productInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("productInfo not sent: %v", body["productInfo"])
	}
	scheduled, ok := productInfo["scheduledAmounts"].([]interface{})
	if !ok || len(scheduled) != 2 {
		t.Fatalf("scheduledAmounts not sent as documented: %v", productInfo["scheduledAmounts"])
	}
	topupInfo, ok := body["topupInfo"].(map[string]interface{})
	if !ok || topupInfo["topupRequest"] != "mimo-topup-001" || topupInfo["topupAmount"] != "30.00" {
		t.Fatalf("topupInfo not sent as documented: %v", body["topupInfo"])
	}
}

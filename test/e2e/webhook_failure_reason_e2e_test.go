//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/utils"
)

func TestWebhookFailureReasons_HTTPServerE2E(t *testing.T) {
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cfg := &config.WaffoConfig{
		APIKey:         "test-api-key",
		PrivateKey:     keyPair.PrivateKey,
		WaffoPublicKey: keyPair.PublicKey,
	}

	reason := "{\"orderFailedCode\":\"K024\",\"orderFailedDescription\":\"payment failed\"}"

	cases := []struct {
		name        string
		eventType   string
		reasonField string
		payload     map[string]interface{}
		register    func(*core.WebhookHandler, *string)
	}{
		{
			name:        "payment failure orderFailedReason",
			eventType:   core.EventPayment,
			reasonField: "orderFailedReason",
			payload: map[string]interface{}{
				"eventType": core.EventPayment,
				"result": map[string]interface{}{
					"paymentRequestId":  "REQ_FAILED",
					"acquiringOrderId":  "ACQ_FAILED",
					"orderStatus":       core.OrderStatusOrderClose,
					"orderAmount":       "100.00",
					"orderCurrency":     "USD",
					"orderFailedReason": reason,
				},
			},
			register: func(handler *core.WebhookHandler, capturedReason *string) {
				handler.OnPayment(func(n *core.PaymentNotification) {
					if n.Result != nil {
						*capturedReason = n.Result.OrderFailedReason.String()
					}
				})
			},
		},
		{
			name:        "refund failure refundFailedReason",
			eventType:   core.EventRefund,
			reasonField: "refundFailedReason",
			payload: map[string]interface{}{
				"eventType": core.EventRefund,
				"result": map[string]interface{}{
					"refundRequestId":        "REFUND_FAILED",
					"acquiringRefundOrderId": "ARF_FAILED",
					"refundStatus":           core.RefundStatusFailed,
					"refundAmount":           "10.00",
					"refundFailedReason":     reason,
				},
			},
			register: func(handler *core.WebhookHandler, capturedReason *string) {
				handler.OnRefund(func(n *core.RefundNotification) {
					if n.Result != nil {
						*capturedReason = n.Result.RefundFailedReason.String()
					}
				})
			},
		},
		{
			name:        "subscription failure failedReason",
			eventType:   core.EventSubscriptionStatus,
			reasonField: "failedReason",
			payload: map[string]interface{}{
				"eventType": core.EventSubscriptionStatus,
				"result": map[string]interface{}{
					"subscriptionId":     "SUB_FAILED",
					"subscriptionStatus": core.SubscriptionStatusClose,
					"failedReason":       reason,
				},
			},
			register: func(handler *core.WebhookHandler, capturedReason *string) {
				handler.OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {
					if n.Result != nil {
						*capturedReason = n.Result.FailedReason.String()
					}
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := core.NewWebhookHandler(cfg)
			capturedReason := ""
			tc.register(handler, &capturedReason)

			webhookTestServer := NewWebhookTestServer(handler, 0)
			httpServer := httptest.NewServer(http.HandlerFunc(webhookTestServer.handleWebhook))
			defer httpServer.Close()

			bodyBytes, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}
			body := string(bodyBytes)
			signature, err := utils.Sign(body, keyPair.PrivateKey)
			if err != nil {
				t.Fatalf("Failed to sign webhook payload: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, httpServer.URL, bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("X-SIGNATURE", signature)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Webhook HTTP request failed: %v", err)
			}
			defer resp.Body.Close()

			responseBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected HTTP 200, got %d: %s", resp.StatusCode, string(responseBody))
			}
			if string(responseBody) != `{"message":"success"}` {
				t.Fatalf("Expected success response body, got %s", string(responseBody))
			}
			if !utils.Verify(string(responseBody), resp.Header.Get("X-SIGNATURE"), keyPair.PublicKey) {
				t.Fatal("Expected response signature to be valid")
			}

			notifications := webhookTestServer.GetNotificationsByType(tc.eventType)
			if len(notifications) != 1 {
				t.Fatalf("Expected exactly one %s notification, got %d", tc.eventType, len(notifications))
			}
			if !notifications[0].HandlerSuccess {
				t.Fatalf("Expected webhook handler success for %s: parsed=%+v", tc.eventType, notifications[0].Parsed)
			}
			if capturedReason != reason {
				t.Fatalf("Expected captured reason %q, got %q", reason, capturedReason)
			}

			resultPayload, ok := notifications[0].Parsed["result"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected parsed result object: %+v", notifications[0].Parsed)
			}
			if parsedReason, ok := resultPayload[tc.reasonField].(string); !ok || parsedReason != reason {
				t.Fatalf("Expected parsed %s %q, got %#v", tc.reasonField, reason, resultPayload[tc.reasonField])
			}
		})
	}
}

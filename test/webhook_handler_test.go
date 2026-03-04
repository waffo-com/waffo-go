package test

import (
	"encoding/json"
	"testing"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/utils"
)

func createTestWebhookHandler(t *testing.T) (*core.WebhookHandler, *utils.KeyPair) {
	// Generate temp key pair for testing
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cfg := &config.WaffoConfig{
		APIKey:         "test-api-key",
		PrivateKey:     keyPair.PrivateKey,
		WaffoPublicKey: keyPair.PublicKey, // Use temp public key for verification
	}

	return core.NewWebhookHandler(cfg), keyPair
}

func TestWebhookHandlerPaymentNotification(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	// Track if handler was called
	handlerCalled := false
	var receivedNotification *core.PaymentNotification

	handler.OnPayment(func(n *core.PaymentNotification) {
		handlerCalled = true
		receivedNotification = n
	})

	// Create test payload (nested structure matching Java SDK)
	payload := map[string]interface{}{
		"eventType": "PAYMENT_NOTIFICATION",
		"result": map[string]interface{}{
			"acquiringOrderId": "ACQ123",
			"merchantOrderId":  "ORDER123",
			"paymentRequestId": "REQ123",
			"orderStatus":      "PAY_SUCCESS",
			"orderAmount":      "100.00",
			"orderCurrency":    "USD",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)

	// Sign payload (simulate Waffo signature)
	signature, _ := utils.Sign(payloadStr, keyPair.PrivateKey)

	// Handle webhook
	result := handler.HandleWebhook(payloadStr, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed, got error: %s", result.Error)
	}

	if !handlerCalled {
		t.Error("Payment handler should be called")
	}

	if receivedNotification == nil {
		t.Fatal("Received notification should not be nil")
	}

	if receivedNotification.Result == nil {
		t.Fatal("Received notification Result should not be nil")
	}

	if receivedNotification.Result.PaymentRequestID != "REQ123" {
		t.Errorf("Expected PaymentRequestID 'REQ123', got '%s'", receivedNotification.Result.PaymentRequestID)
	}

	if receivedNotification.Result.OrderStatus != "PAY_SUCCESS" {
		t.Errorf("Expected OrderStatus 'PAY_SUCCESS', got '%s'", receivedNotification.Result.OrderStatus)
	}
}

func TestWebhookHandlerRefundNotification(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	handlerCalled := false
	handler.OnRefund(func(n *core.RefundNotification) {
		handlerCalled = true
		if n.Result == nil {
			t.Fatal("Refund notification Result should not be nil")
		}
		if n.Result.RefundStatus != "ORDER_FULLY_REFUNDED" {
			t.Errorf("Expected RefundStatus 'ORDER_FULLY_REFUNDED', got '%s'", n.Result.RefundStatus)
		}
	})

	payload := map[string]interface{}{
		"eventType": "REFUND_NOTIFICATION",
		"result": map[string]interface{}{
			"acquiringRefundOrderId": "REF123",
			"refundRequestId":        "REFUND_REQ123",
			"refundStatus":           "ORDER_FULLY_REFUNDED",
			"refundAmount":           "100.00",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)
	signature, _ := utils.Sign(payloadStr, keyPair.PrivateKey)

	result := handler.HandleWebhook(payloadStr, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed: %s", result.Error)
	}

	if !handlerCalled {
		t.Error("Refund handler should be called")
	}
}

func TestWebhookHandlerSubscriptionStatusNotification(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	handlerCalled := false
	handler.OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {
		handlerCalled = true
		if n.Result == nil {
			t.Fatal("Subscription status notification Result should not be nil")
		}
		if n.Result.SubscriptionStatus != "ACTIVE" {
			t.Errorf("Expected SubscriptionStatus 'ACTIVE', got '%s'", n.Result.SubscriptionStatus)
		}
	})

	payload := map[string]interface{}{
		"eventType": "SUBSCRIPTION_STATUS_NOTIFICATION",
		"result": map[string]interface{}{
			"subscriptionId":     "SUB123",
			"subscriptionStatus": "ACTIVE",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)
	signature, _ := utils.Sign(payloadStr, keyPair.PrivateKey)

	result := handler.HandleWebhook(payloadStr, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed: %s", result.Error)
	}

	if !handlerCalled {
		t.Error("Subscription status handler should be called")
	}
}

func TestWebhookHandlerSubscriptionPaymentFallback(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	// Only register subscription payment handler (not subscription status handler)
	// SUBSCRIPTION_STATUS_NOTIFICATION should fall back to subscription payment handler
	handlerCalled := false
	handler.OnSubscriptionPayment(func(n *core.SubscriptionStatusNotification) {
		handlerCalled = true
		if n.Result == nil {
			t.Fatal("Subscription payment notification Result should not be nil")
		}
		if n.Result.SubscriptionStatus != "ACTIVE" {
			t.Errorf("Expected SubscriptionStatus 'ACTIVE', got '%s'", n.Result.SubscriptionStatus)
		}
	})

	payload := map[string]interface{}{
		"eventType": "SUBSCRIPTION_STATUS_NOTIFICATION",
		"result": map[string]interface{}{
			"subscriptionId":     "SUB456",
			"subscriptionStatus": "ACTIVE",
			"currency":           "HKD",
			"amount":             "50.00",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)
	signature, _ := utils.Sign(payloadStr, keyPair.PrivateKey)

	result := handler.HandleWebhook(payloadStr, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed: %s", result.Error)
	}

	if !handlerCalled {
		t.Error("Subscription payment handler should be called as fallback")
	}
}

func TestWebhookHandlerSubscriptionStatusPriorityOverPayment(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	// Register both handlers - status handler should take priority
	statusCalled := false
	paymentCalled := false
	handler.OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {
		statusCalled = true
	}).OnSubscriptionPayment(func(n *core.SubscriptionStatusNotification) {
		paymentCalled = true
	})

	payload := map[string]interface{}{
		"eventType": "SUBSCRIPTION_STATUS_NOTIFICATION",
		"result": map[string]interface{}{
			"subscriptionId":     "SUB789",
			"subscriptionStatus": "ACTIVE",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)
	signature, _ := utils.Sign(payloadStr, keyPair.PrivateKey)

	result := handler.HandleWebhook(payloadStr, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed: %s", result.Error)
	}

	if !statusCalled {
		t.Error("Subscription status handler should be called (has priority)")
	}

	if paymentCalled {
		t.Error("Subscription payment handler should NOT be called when status handler is registered")
	}
}

func TestWebhookHandlerInvalidSignature(t *testing.T) {
	handler, _ := createTestWebhookHandler(t)

	payload := `{"eventType":"PAYMENT_NOTIFICATION"}`

	// Invalid signature
	result := handler.HandleWebhook(payload, "invalid-signature")

	if result.Success {
		t.Error("HandleWebhook() should fail for invalid signature")
	}

	if result.Error != "invalid signature" {
		t.Errorf("Expected error 'invalid signature', got '%s'", result.Error)
	}
}

func TestWebhookHandlerMissingSignature(t *testing.T) {
	handler, _ := createTestWebhookHandler(t)

	payload := `{"eventType":"PAYMENT_NOTIFICATION"}`

	// Missing signature
	result := handler.HandleWebhook(payload, "")

	if result.Success {
		t.Error("HandleWebhook() should fail for missing signature")
	}

	if result.Error != "missing signature" {
		t.Errorf("Expected error 'missing signature', got '%s'", result.Error)
	}
}

func TestWebhookHandlerUnknownEventType(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	payload := `{"eventType":"UNKNOWN_EVENT"}`
	signature, _ := utils.Sign(payload, keyPair.PrivateKey)

	result := handler.HandleWebhook(payload, signature)

	if result.Success {
		t.Error("HandleWebhook() should fail for unknown event type")
	}

	if result.Error != "unknown event type: UNKNOWN_EVENT" {
		t.Errorf("Expected error about unknown event type, got '%s'", result.Error)
	}
}

func TestWebhookHandlerResponseSigning(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	// Test success response
	body, signature := handler.BuildSuccessResponse()

	if body != `{"message":"success"}` {
		t.Errorf("Expected success response body, got '%s'", body)
	}

	// Verify response signature
	if !utils.Verify(body, signature, keyPair.PublicKey) {
		t.Error("Success response signature should be valid")
	}

	// Test failed response
	body, signature = handler.BuildFailedResponse("test error")

	if body != `{"message":"failed"}` {
		t.Errorf("Expected failed response body, got '%s'", body)
	}

	// Verify response signature
	if !utils.Verify(body, signature, keyPair.PublicKey) {
		t.Error("Failed response signature should be valid")
	}
}

func TestWebhookHandlerSubscriptionChangeNotification(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	handlerCalled := false
	var receivedNotification *core.SubscriptionChangeNotification

	handler.OnSubscriptionChange(func(n *core.SubscriptionChangeNotification) {
		handlerCalled = true
		receivedNotification = n
	})

	payload := map[string]interface{}{
		"eventType": "SUBSCRIPTION_CHANGE_NOTIFICATION",
		"result": map[string]interface{}{
			"subscriptionRequest":       "new-sub-req-001",
			"originSubscriptionRequest": "orig-sub-req-001",
			"merchantSubscriptionId":    "MERCHANT_SUB_001",
			"subscriptionId":            "waffo-sub-12345",
			"subscriptionChangeStatus":  "SUCCESS",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)
	signature, _ := utils.Sign(payloadStr, keyPair.PrivateKey)

	result := handler.HandleWebhook(payloadStr, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed: %s", result.Error)
	}

	if !handlerCalled {
		t.Error("Subscription change handler should be called")
	}

	if receivedNotification == nil {
		t.Fatal("Received notification should not be nil")
	}

	if receivedNotification.Result == nil {
		t.Fatal("Received notification Result should not be nil")
	}

	if receivedNotification.Result.SubscriptionChangeStatus != "SUCCESS" {
		t.Errorf("Expected SubscriptionChangeStatus 'SUCCESS', got '%s'", receivedNotification.Result.SubscriptionChangeStatus)
	}

	if receivedNotification.Result.SubscriptionRequest != "new-sub-req-001" {
		t.Errorf("Expected SubscriptionRequest 'new-sub-req-001', got '%s'", receivedNotification.Result.SubscriptionRequest)
	}

	if receivedNotification.Result.OriginSubscriptionRequest != "orig-sub-req-001" {
		t.Errorf("Expected OriginSubscriptionRequest 'orig-sub-req-001', got '%s'", receivedNotification.Result.OriginSubscriptionRequest)
	}
}

func TestWebhookHandlerChaining(t *testing.T) {
	handler, _ := createTestWebhookHandler(t)

	// Test that handlers can be chained
	result := handler.
		OnPayment(func(n *core.PaymentNotification) {}).
		OnRefund(func(n *core.RefundNotification) {}).
		OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {}).
		OnSubscriptionPayment(func(n *core.SubscriptionStatusNotification) {}).
		OnSubscriptionPeriodChanged(func(n *core.SubscriptionPeriodChangedNotification) {}).
		OnSubscriptionChange(func(n *core.SubscriptionChangeNotification) {})

	if result == nil {
		t.Error("Handler methods should return handler for chaining")
	}
}

func TestWebhookHandlerNoRegisteredHandler(t *testing.T) {
	handler, keyPair := createTestWebhookHandler(t)

	// Don't register any handlers
	payload := `{"eventType":"PAYMENT_NOTIFICATION"}`
	signature, _ := utils.Sign(payload, keyPair.PrivateKey)

	// Should succeed even without handler (just acknowledge)
	result := handler.HandleWebhook(payload, signature)

	if !result.Success {
		t.Errorf("HandleWebhook() should succeed even without registered handler: %s", result.Error)
	}
}

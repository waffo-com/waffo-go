package vectors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/waffo-com/waffo-go/v2/config"
	"github.com/waffo-com/waffo-go/v2/core"
	waffoerrors "github.com/waffo-com/waffo-go/v2/errors"
	"github.com/waffo-com/waffo-go/v2/types/order"
	"github.com/waffo-com/waffo-go/v2/types/subscription"
	"github.com/waffo-com/waffo-go/v2/utils"
)

// openValuesRegistrationErrorCode is the shared configuration error code from
// sdk-spec/ERROR_HANDLING.md and sdk-spec/WEBHOOK_HANDLER.md.
const openValuesRegistrationErrorCode = "S0006"

type openValuesVector struct {
	Module            string                 `json:"module"`
	BusinessValues    []string               `json:"businessValues"`
	KnownEvents       []openValuesKnownEvent `json:"knownEvents"`
	FailureEvents     []openValuesKnownEvent `json:"failureEvents"`
	CustomEvent       openValuesCustomEvent  `json:"customEvent"`
	InvalidEventNames []string               `json:"invalidEventNames"`
}

type openValuesKnownEvent struct {
	EventType string                 `json:"eventType"`
	Result    map[string]interface{} `json:"result"`
}

type openValuesCustomEvent struct {
	EventType   string                 `json:"eventType"`
	Result      map[string]interface{} `json:"result"`
	FutureField map[string]interface{} `json:"futureField"`
}

type openValuesKeys struct {
	privateKey string
	publicKey  string
}

// loadOpenValuesVector reads the registered monorepo vector consumed by every SDK.
func loadOpenValuesVector(t *testing.T) openValuesVector {
	t.Helper()
	data, err := os.ReadFile("../../sdk-spec/test-vectors/open-values.json")
	if err != nil {
		t.Fatalf("read open-values.json: %v", err)
	}

	var vector openValuesVector
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatalf("parse open-values.json: %v", err)
	}
	if vector.Module != "open-values" {
		t.Fatalf("unexpected open-values vector module: %q", vector.Module)
	}
	if len(vector.BusinessValues) == 0 || len(vector.KnownEvents) != 5 {
		t.Fatalf("open-values vector is missing cases: %+v", vector)
	}
	if vector.CustomEvent.EventType == "" || len(vector.InvalidEventNames) == 0 {
		t.Fatalf("open-values vector is missing custom event cases: %+v", vector)
	}
	return vector
}

// newOpenValuesWebhookHandler reuses the existing RSA signing vector fixture.
func newOpenValuesWebhookHandler(t *testing.T) (*core.WebhookHandler, openValuesKeys) {
	t.Helper()
	testKeys := loadRSAVector(t).TestKeys
	keys := openValuesKeys{privateKey: testKeys.PrivateKey, publicKey: testKeys.PublicKey}
	handler := core.NewWebhookHandler(&config.WaffoConfig{
		APIKey:         "test-api-key",
		PrivateKey:     keys.privateKey,
		WaffoPublicKey: keys.publicKey,
	})
	return handler, keys
}

func signOpenValuesPayload(t *testing.T, body string, keys openValuesKeys) string {
	t.Helper()
	signature, err := utils.Sign(body, keys.privateKey)
	if err != nil {
		t.Fatalf("sign open-values payload: %v", err)
	}
	return signature
}

func requireOpenValuesSuccess(t *testing.T, result *core.WebhookResult, keys openValuesKeys, context string) {
	t.Helper()
	if result == nil {
		t.Fatalf("%s: webhook result is nil", context)
	}
	if !result.Success {
		t.Fatalf("%s: expected signed success, got error %q with body %s", context, result.Error, result.ResponseBody)
	}
	if result.ResponseBody != `{"message":"success"}` {
		t.Fatalf("%s: expected exact success body, got %q", context, result.ResponseBody)
	}
	if !utils.Verify(result.ResponseBody, result.ResponseSignature, keys.publicKey) {
		t.Fatalf("%s: success response signature does not verify", context)
	}
}

func requireOpenValuesFailure(t *testing.T, result *core.WebhookResult, keys openValuesKeys, context string) {
	t.Helper()
	if result == nil {
		t.Fatalf("%s: webhook result is nil", context)
	}
	if result.Success {
		t.Fatalf("%s: expected signed failure, got success", context)
	}
	if result.ResponseBody != `{"message":"failed"}` {
		t.Fatalf("%s: expected exact failed body, got %q", context, result.ResponseBody)
	}
	if !utils.Verify(result.ResponseBody, result.ResponseSignature, keys.publicKey) {
		t.Fatalf("%s: failed response signature does not verify", context)
	}
}

func requireOpenValuesS0006(t *testing.T, err error, context string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected WaffoError %s, got nil", context, openValuesRegistrationErrorCode)
	}

	var waffoErr *waffoerrors.WaffoError
	if !errors.As(err, &waffoErr) {
		t.Fatalf("%s: expected *WaffoError, got %T: %v", context, err, err)
	}
	if waffoErr.ErrorCode != openValuesRegistrationErrorCode {
		t.Fatalf("%s: expected error code %s, got %s (%v)",
			context, openValuesRegistrationErrorCode, waffoErr.ErrorCode, err)
	}
}

func openValuesString(t *testing.T, values map[string]interface{}, key string, context string) string {
	t.Helper()
	value, ok := values[key]
	if !ok {
		t.Fatalf("%s: vector is missing %q", context, key)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s: vector %q is not a string: %#v", context, key, value)
	}
	return text
}

func openValuesPaymentDetailStatus(t *testing.T, values map[string]interface{}, context string) string {
	t.Helper()
	details, ok := values["paymentDetails"].([]interface{})
	if !ok || len(details) != 1 {
		t.Fatalf("%s: vector paymentDetails is not a single-element array: %#v", context, values["paymentDetails"])
	}
	detail, ok := details[0].(map[string]interface{})
	if !ok {
		t.Fatalf("%s: vector paymentDetails[0] is not an object: %#v", context, details[0])
	}
	return openValuesString(t, detail, "orderStatus", context+".paymentDetails[0]")
}

func openValuesKnownEventBody(t *testing.T, event openValuesKnownEvent) string {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"eventType": event.EventType,
		"result":    event.Result,
	})
	if err != nil {
		t.Fatalf("marshal %s body: %v", event.EventType, err)
	}
	return string(body)
}

func openValuesCustomEventPayload(custom openValuesCustomEvent) map[string]interface{} {
	return map[string]interface{}{
		"eventType":   custom.EventType,
		"result":      custom.Result,
		"futureField": custom.FutureField,
	}
}

// openValuesCustomEventBody keeps real JSON whitespace so the Go handler can be
// checked for byte-for-byte json.RawMessage delivery.
func openValuesCustomEventBody(t *testing.T, custom openValuesCustomEvent) string {
	t.Helper()
	compact, err := json.Marshal(openValuesCustomEventPayload(custom))
	if err != nil {
		t.Fatalf("marshal custom event payload: %v", err)
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, compact, "  ", "  "); err != nil {
		t.Fatalf("indent custom event payload: %v", err)
	}
	body := indented.String()
	if !strings.Contains(body, "\n") || !strings.Contains(body, "  ") {
		t.Fatalf("custom event body must contain whitespace, got %q", body)
	}
	return body
}

func openValuesCustomEventBodyWithName(t *testing.T, custom openValuesCustomEvent, name string) string {
	t.Helper()
	payload := openValuesCustomEventPayload(custom)
	payload["eventType"] = name
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal custom event body for %q: %v", name, err)
	}
	return string(body)
}

func openValuesHandleWebhookSafely(
	handler *core.WebhookHandler,
	body string,
	signature string,
) (result *core.WebhookResult, panicked interface{}) {
	defer func() {
		panicked = recover()
	}()
	result = handler.HandleWebhook(body, signature)
	return result, nil
}

func TestOpenValuesKnownEventsRouteNewBusinessValues(t *testing.T) {
	vector := loadOpenValuesVector(t)

	builtinNames := map[string]bool{
		core.EventPayment:                   true,
		core.EventRefund:                    true,
		core.EventSubscriptionStatus:        true,
		core.EventSubscriptionPeriodChanged: true,
		core.EventSubscriptionChange:        true,
	}
	for _, event := range vector.KnownEvents {
		if !builtinNames[event.EventType] {
			t.Fatalf("vector known event %q is not a builtin event", event.EventType)
		}
	}

	for _, event := range vector.KnownEvents {
		event := event
		t.Run(event.EventType, func(t *testing.T) {
			handler, keys := newOpenValuesWebhookHandler(t)
			body := openValuesKnownEventBody(t, event)
			signature := signOpenValuesPayload(t, body, keys)
			expectedID := ""
			expectedStatus := ""

			switch event.EventType {
			case core.EventPayment:
				expectedID = openValuesString(t, event.Result, "acquiringOrderId", event.EventType)
				expectedStatus = openValuesString(t, event.Result, "orderStatus", event.EventType)
				calls := 0
				var received *core.PaymentNotification
				handler.OnPayment(func(notification *core.PaymentNotification) {
					calls++
					received = notification
				})

				result := handler.HandleWebhook(body, signature)
				requireOpenValuesSuccess(t, result, keys, event.EventType)
				if calls != 1 || received == nil || received.Result == nil {
					t.Fatalf("%s: payment handler calls=%d notification=%+v", event.EventType, calls, received)
				}
				if received.Result.AcquiringOrderID != expectedID {
					t.Errorf("%s: expected acquiringOrderId %q, got %q",
						event.EventType, expectedID, received.Result.AcquiringOrderID)
				}
				if received.Result.OrderStatus != expectedStatus {
					t.Errorf("%s: expected orderStatus %q, got %q",
						event.EventType, expectedStatus, received.Result.OrderStatus)
				}

			case core.EventRefund:
				expectedID = openValuesString(t, event.Result, "acquiringRefundOrderId", event.EventType)
				expectedStatus = openValuesString(t, event.Result, "refundStatus", event.EventType)
				calls := 0
				var received *core.RefundNotification
				handler.OnRefund(func(notification *core.RefundNotification) {
					calls++
					received = notification
				})

				result := handler.HandleWebhook(body, signature)
				requireOpenValuesSuccess(t, result, keys, event.EventType)
				if calls != 1 || received == nil || received.Result == nil {
					t.Fatalf("%s: refund handler calls=%d notification=%+v", event.EventType, calls, received)
				}
				if received.Result.AcquiringRefundOrderID != expectedID {
					t.Errorf("%s: expected acquiringRefundOrderId %q, got %q",
						event.EventType, expectedID, received.Result.AcquiringRefundOrderID)
				}
				if received.Result.RefundStatus != expectedStatus {
					t.Errorf("%s: expected refundStatus %q, got %q",
						event.EventType, expectedStatus, received.Result.RefundStatus)
				}

			case core.EventSubscriptionStatus:
				expectedID = openValuesString(t, event.Result, "subscriptionId", event.EventType)
				expectedStatus = openValuesString(t, event.Result, "subscriptionStatus", event.EventType)
				detailStatus := openValuesPaymentDetailStatus(t, event.Result, event.EventType)
				if expectedStatus != "ACTIVE" || detailStatus != "INIT" {
					t.Fatalf("%s: vector must carry ACTIVE with INIT payment detail, got %q/%q",
						event.EventType, expectedStatus, detailStatus)
				}
				calls := 0
				var received *core.SubscriptionStatusNotification
				handler.OnSubscriptionStatus(func(notification *core.SubscriptionStatusNotification) {
					calls++
					received = notification
				})

				result := handler.HandleWebhook(body, signature)
				requireOpenValuesSuccess(t, result, keys, event.EventType)
				if calls != 1 || received == nil || received.Result == nil {
					t.Fatalf("%s: subscription handler calls=%d notification=%+v", event.EventType, calls, received)
				}
				if received.Result.SubscriptionID != expectedID {
					t.Errorf("%s: expected subscriptionId %q, got %q",
						event.EventType, expectedID, received.Result.SubscriptionID)
				}
				if received.Result.SubscriptionStatus != expectedStatus {
					t.Errorf("%s: expected subscriptionStatus %q, got %q",
						event.EventType, expectedStatus, received.Result.SubscriptionStatus)
				}
				if len(received.Result.PaymentDetails) != 1 {
					t.Fatalf("%s: expected one payment detail, got %d",
						event.EventType, len(received.Result.PaymentDetails))
				}
				if received.Result.PaymentDetails[0]["orderStatus"] != detailStatus {
					t.Errorf("%s: expected paymentDetails[0].orderStatus %q, got %#v",
						event.EventType, detailStatus, received.Result.PaymentDetails[0]["orderStatus"])
				}

			case core.EventSubscriptionPeriodChanged:
				expectedID = openValuesString(t, event.Result, "subscriptionId", event.EventType)
				expectedStatus = openValuesString(t, event.Result, "subscriptionStatus", event.EventType)
				detailStatus := openValuesPaymentDetailStatus(t, event.Result, event.EventType)
				calls := 0
				var received *core.SubscriptionPeriodChangedNotification
				handler.OnSubscriptionPeriodChanged(func(notification *core.SubscriptionPeriodChangedNotification) {
					calls++
					received = notification
				})

				result := handler.HandleWebhook(body, signature)
				requireOpenValuesSuccess(t, result, keys, event.EventType)
				if calls != 1 || received == nil || received.Result == nil {
					t.Fatalf("%s: period handler calls=%d notification=%+v", event.EventType, calls, received)
				}
				if received.Result.SubscriptionID != expectedID {
					t.Errorf("%s: expected subscriptionId %q, got %q",
						event.EventType, expectedID, received.Result.SubscriptionID)
				}
				if received.Result.SubscriptionStatus != expectedStatus {
					t.Errorf("%s: expected subscriptionStatus %q, got %q",
						event.EventType, expectedStatus, received.Result.SubscriptionStatus)
				}
				if len(received.Result.PaymentDetails) != 1 {
					t.Fatalf("%s: expected one payment detail, got %d",
						event.EventType, len(received.Result.PaymentDetails))
				}
				if received.Result.PaymentDetails[0]["orderStatus"] != detailStatus {
					t.Errorf("%s: expected paymentDetails[0].orderStatus %q, got %#v",
						event.EventType, detailStatus, received.Result.PaymentDetails[0]["orderStatus"])
				}

			case core.EventSubscriptionChange:
				expectedID = openValuesString(t, event.Result, "subscriptionId", event.EventType)
				expectedStatus = openValuesString(t, event.Result, "subscriptionChangeStatus", event.EventType)
				calls := 0
				var received *core.SubscriptionChangeNotification
				handler.OnSubscriptionChange(func(notification *core.SubscriptionChangeNotification) {
					calls++
					received = notification
				})

				result := handler.HandleWebhook(body, signature)
				requireOpenValuesSuccess(t, result, keys, event.EventType)
				if calls != 1 || received == nil || received.Result == nil {
					t.Fatalf("%s: change handler calls=%d notification=%+v", event.EventType, calls, received)
				}
				if received.Result.SubscriptionID != expectedID {
					t.Errorf("%s: expected subscriptionId %q, got %q",
						event.EventType, expectedID, received.Result.SubscriptionID)
				}
				if received.Result.SubscriptionChangeStatus != expectedStatus {
					t.Errorf("%s: expected subscriptionChangeStatus %q, got %q",
						event.EventType, expectedStatus, received.Result.SubscriptionChangeStatus)
				}

			default:
				t.Fatalf("unhandled known event %q", event.EventType)
			}
		})
	}
}

func TestOpenValuesBuiltinEventsWithoutHandlerSucceed(t *testing.T) {
	vector := loadOpenValuesVector(t)

	for _, event := range vector.KnownEvents {
		event := event
		t.Run(event.EventType, func(t *testing.T) {
			handler, keys := newOpenValuesWebhookHandler(t)
			body := openValuesKnownEventBody(t, event)
			result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
			requireOpenValuesSuccess(t, result, keys, event.EventType+" without handler")
		})
	}
}

func TestOpenValuesSubscriptionStatusPriorityAndFallback(t *testing.T) {
	vector := loadOpenValuesVector(t)

	var statusEvent openValuesKnownEvent
	for _, event := range vector.KnownEvents {
		if event.EventType == core.EventSubscriptionStatus {
			statusEvent = event
			break
		}
	}
	if statusEvent.EventType == "" {
		t.Fatalf("vector is missing %s", core.EventSubscriptionStatus)
	}
	body := openValuesKnownEventBody(t, statusEvent)

	t.Run("status handler has priority", func(t *testing.T) {
		handler, keys := newOpenValuesWebhookHandler(t)
		statusCalls := 0
		paymentCalls := 0
		handler.
			OnSubscriptionStatus(func(*core.SubscriptionStatusNotification) { statusCalls++ }).
			OnSubscriptionPayment(func(*core.SubscriptionStatusNotification) { paymentCalls++ })

		result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
		requireOpenValuesSuccess(t, result, keys, "subscription status priority")
		if statusCalls != 1 {
			t.Errorf("expected subscription status handler once, got %d", statusCalls)
		}
		if paymentCalls != 0 {
			t.Errorf("expected payment fallback zero calls, got %d", paymentCalls)
		}
	})

	t.Run("payment handler is fallback", func(t *testing.T) {
		handler, keys := newOpenValuesWebhookHandler(t)
		paymentCalls := 0
		handler.OnSubscriptionPayment(func(*core.SubscriptionStatusNotification) { paymentCalls++ })

		result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
		requireOpenValuesSuccess(t, result, keys, "subscription payment fallback")
		if paymentCalls != 1 {
			t.Errorf("expected payment fallback once, got %d", paymentCalls)
		}
	})
}

func TestOpenValuesCustomEventRequiresExactRegisteredName(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	t.Run("unregistered custom event fails with zero calls", func(t *testing.T) {
		handler, keys := newOpenValuesWebhookHandler(t)
		sentinelCalls := 0
		if _, err := handler.OnEvent(custom.EventType+"_OTHER", func(json.RawMessage) error {
			sentinelCalls++
			return nil
		}); err != nil {
			t.Fatalf("register sentinel custom event: %v", err)
		}

		result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
		requireOpenValuesFailure(t, result, keys, "unregistered custom event")
		if sentinelCalls != 0 {
			t.Errorf("expected zero custom handler calls, got %d", sentinelCalls)
		}
	})

	t.Run("name casing is not matched", func(t *testing.T) {
		handler, keys := newOpenValuesWebhookHandler(t)
		registeredCalls := 0
		if _, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
			registeredCalls++
			return nil
		}); err != nil {
			t.Fatalf("register custom event: %v", err)
		}

		lowerName := strings.ToLower(custom.EventType)
		if lowerName == custom.EventType {
			t.Fatalf("vector custom event %q must contain uppercase characters", custom.EventType)
		}
		mismatchedBody := openValuesCustomEventBodyWithName(t, custom, lowerName)
		result := handler.HandleWebhook(mismatchedBody, signOpenValuesPayload(t, mismatchedBody, keys))
		requireOpenValuesFailure(t, result, keys, "custom event name with different casing")
		if registeredCalls != 0 {
			t.Errorf("expected zero custom handler calls for case mismatch, got %d", registeredCalls)
		}

		result = handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
		requireOpenValuesSuccess(t, result, keys, "exact custom event name after case mismatch")
		if registeredCalls != 1 {
			t.Errorf("expected exact registration to stay active, got %d calls", registeredCalls)
		}
	})
}

func TestOpenValuesRegisteredCustomEventReceivesExactRawPayload(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	handler, keys := newOpenValuesWebhookHandler(t)
	var received []json.RawMessage
	returned, err := handler.OnEvent(custom.EventType, func(raw json.RawMessage) error {
		received = append(received, append(json.RawMessage(nil), raw...))
		return nil
	})
	if err != nil {
		t.Fatalf("register custom event: %v", err)
	}
	if returned != handler {
		t.Fatalf("OnEvent must return the same handler instance")
	}

	result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
	requireOpenValuesSuccess(t, result, keys, "registered custom event")
	if len(received) != 1 {
		t.Fatalf("expected one custom handler call, got %d", len(received))
	}
	if string(received[0]) != body {
		t.Fatalf("custom handler raw payload changed: got %q want %q", string(received[0]), body)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(received[0], &decoded); err != nil {
		t.Fatalf("decode raw custom payload: %v", err)
	}
	if openValuesString(t, decoded, "eventType", "custom payload") != custom.EventType {
		t.Errorf("expected eventType %q, got %#v", custom.EventType, decoded["eventType"])
	}
	decodedResult, ok := decoded["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("custom payload result is not an object: %#v", decoded["result"])
	}
	if decodedResult["id"] != custom.Result["id"] || decodedResult["status"] != custom.Result["status"] {
		t.Errorf("custom payload result changed: got %#v want %#v", decodedResult, custom.Result)
	}
	futureField, ok := decoded["futureField"].(map[string]interface{})
	if !ok {
		t.Fatalf("custom payload futureField is not an object: %#v", decoded["futureField"])
	}
	if futureField["kept"] != custom.FutureField["kept"] {
		t.Errorf("custom payload futureField changed: got %#v want %#v", futureField, custom.FutureField)
	}
}

func TestOpenValuesDuplicateCustomRegistrationReplacesPrevious(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	handler, keys := newOpenValuesWebhookHandler(t)
	firstCalls := 0
	secondCalls := 0
	if _, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
		firstCalls++
		return nil
	}); err != nil {
		t.Fatalf("register first custom handler: %v", err)
	}
	if _, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
		secondCalls++
		return nil
	}); err != nil {
		t.Fatalf("register replacement custom handler: %v", err)
	}

	result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
	requireOpenValuesSuccess(t, result, keys, "duplicate custom registration")
	if firstCalls != 0 {
		t.Errorf("expected replaced handler zero calls, got %d", firstCalls)
	}
	if secondCalls != 1 {
		t.Errorf("expected replacement handler one call, got %d", secondCalls)
	}
}

func TestOpenValuesCustomRegistrationsAreInstanceScoped(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	registeredHandler, registeredKeys := newOpenValuesWebhookHandler(t)
	calls := 0
	if _, err := registeredHandler.OnEvent(custom.EventType, func(json.RawMessage) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("register custom event: %v", err)
	}

	idleHandler, idleKeys := newOpenValuesWebhookHandler(t)
	idleResult := idleHandler.HandleWebhook(body, signOpenValuesPayload(t, body, idleKeys))
	requireOpenValuesFailure(t, idleResult, idleKeys, "custom event on isolated handler instance")
	if calls != 0 {
		t.Errorf("expected isolated handler zero calls, got %d", calls)
	}

	registeredResult := registeredHandler.HandleWebhook(body, signOpenValuesPayload(t, body, registeredKeys))
	requireOpenValuesSuccess(t, registeredResult, registeredKeys, "custom event on registered handler instance")
	if calls != 1 {
		t.Errorf("expected registered handler one call, got %d", calls)
	}
}

func TestOpenValuesSignatureVerificationPrecedesCustomCallback(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	handler, keys := newOpenValuesWebhookHandler(t)
	calls := 0
	if _, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("register custom event: %v", err)
	}

	missing := handler.HandleWebhook(body, "")
	requireOpenValuesFailure(t, missing, keys, "missing signature")
	if calls != 0 {
		t.Errorf("expected zero custom calls for missing signature, got %d", calls)
	}

	invalid := handler.HandleWebhook(body, "invalid-signature")
	requireOpenValuesFailure(t, invalid, keys, "invalid signature")
	if calls != 0 {
		t.Errorf("expected zero custom calls for invalid signature, got %d", calls)
	}

	malformed := "{not-json"
	malformedResult := handler.HandleWebhook(malformed, signOpenValuesPayload(t, malformed, keys))
	requireOpenValuesFailure(t, malformedResult, keys, "valid signature with invalid JSON")
	if calls != 0 {
		t.Errorf("expected zero custom calls for invalid JSON, got %d", calls)
	}
}

func TestOpenValuesCustomCallbackErrorAndPanicReturnSignedFailure(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	t.Run("error", func(t *testing.T) {
		handler, keys := newOpenValuesWebhookHandler(t)
		calls := 0
		if _, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
			calls++
			return errors.New("custom handler failed")
		}); err != nil {
			t.Fatalf("register custom event: %v", err)
		}

		result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
		requireOpenValuesFailure(t, result, keys, "custom handler error")
		if calls != 1 {
			t.Errorf("expected one custom handler call, got %d", calls)
		}
	})

	t.Run("panic", func(t *testing.T) {
		handler, keys := newOpenValuesWebhookHandler(t)
		calls := 0
		if _, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
			calls++
			panic("custom handler panic")
		}); err != nil {
			t.Fatalf("register custom event: %v", err)
		}

		result, panicked := openValuesHandleWebhookSafely(
			handler,
			body,
			signOpenValuesPayload(t, body, keys),
		)
		if panicked != nil {
			t.Fatalf("panic escaped HandleWebhook: %v", panicked)
		}
		requireOpenValuesFailure(t, result, keys, "custom handler panic")
		if calls != 1 {
			t.Errorf("expected one custom handler call, got %d", calls)
		}
	})
}

func TestOpenValuesOnEventRejectsInvalidConfiguration(t *testing.T) {
	vector := loadOpenValuesVector(t)
	custom := vector.CustomEvent
	body := openValuesCustomEventBody(t, custom)

	handler, keys := newOpenValuesWebhookHandler(t)
	customCalls := 0
	returned, err := handler.OnEvent(custom.EventType, func(json.RawMessage) error {
		customCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("register custom event: %v", err)
	}
	if returned != handler {
		t.Fatalf("OnEvent must return the same handler instance")
	}

	returned, err = handler.OnEvent(custom.EventType, nil)
	requireOpenValuesS0006(t, err, "nil handler")
	if returned != nil {
		t.Errorf("expected nil handler instance for rejected nil handler")
	}

	returned, err = handler.OnEvent(custom.EventType+"_NEW", nil)
	requireOpenValuesS0006(t, err, "nil handler for new name")
	if returned != nil {
		t.Errorf("expected nil handler instance for rejected nil handler")
	}

	for _, name := range vector.InvalidEventNames {
		name := name
		returned, err := handler.OnEvent(name, func(json.RawMessage) error { return nil })
		requireOpenValuesS0006(t, err, fmt.Sprintf("invalid event name %q", name))
		if returned != nil {
			t.Errorf("invalid event name %q: expected nil handler instance", name)
		}
	}

	for _, event := range vector.KnownEvents {
		returned, err := handler.OnEvent(event.EventType, func(json.RawMessage) error { return nil })
		requireOpenValuesS0006(t, err, "builtin event "+event.EventType)
		if returned != nil {
			t.Errorf("builtin event %q: expected nil handler instance", event.EventType)
		}
	}

	// The original custom mapping must survive every rejected registration.
	result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
	requireOpenValuesSuccess(t, result, keys, "custom mapping after rejected registrations")
	if customCalls != 1 {
		t.Errorf("expected preserved custom handler one call, got %d", customCalls)
	}
}

func TestOpenValuesBusinessModelsPreserveStrings(t *testing.T) {
	vector := loadOpenValuesVector(t)

	for _, value := range vector.BusinessValues {
		value := value
		t.Run(fmt.Sprintf("%q", value), func(t *testing.T) {
			statusJSON, err := json.Marshal(map[string]string{"orderStatus": value})
			if err != nil {
				t.Fatalf("marshal orderStatus: %v", err)
			}

			var detail subscription.PaymentDetail
			if err := json.Unmarshal(statusJSON, &detail); err != nil {
				t.Fatalf("parse subscription.PaymentDetail: %v", err)
			}
			if detail.OrderStatus != value {
				t.Errorf("subscription.PaymentDetail.OrderStatus: got %q want %q", detail.OrderStatus, value)
			}
			detailJSON, err := json.Marshal(detail)
			if err != nil {
				t.Fatalf("serialize subscription.PaymentDetail: %v", err)
			}
			var detailFields map[string]interface{}
			if err := json.Unmarshal(detailJSON, &detailFields); err != nil {
				t.Fatalf("decode serialized subscription.PaymentDetail: %v", err)
			}
			if detailFields["orderStatus"] != value {
				t.Errorf("serialized subscription.PaymentDetail.orderStatus: got %#v want %q",
					detailFields["orderStatus"], value)
			}

			var created order.CreateOrderData
			if err := json.Unmarshal(statusJSON, &created); err != nil {
				t.Fatalf("parse order.CreateOrderData: %v", err)
			}
			if created.OrderStatus != value {
				t.Errorf("order.CreateOrderData.OrderStatus: got %q want %q", created.OrderStatus, value)
			}
			createdJSON, err := json.Marshal(created)
			if err != nil {
				t.Fatalf("serialize order.CreateOrderData: %v", err)
			}
			var createdFields map[string]interface{}
			if err := json.Unmarshal(createdJSON, &createdFields); err != nil {
				t.Fatalf("decode serialized order.CreateOrderData: %v", err)
			}
			if createdFields["orderStatus"] != value {
				t.Errorf("serialized order.CreateOrderData.orderStatus: got %#v want %q",
					createdFields["orderStatus"], value)
			}

			productJSON, err := json.Marshal(map[string]string{
				"description":     "open-values",
				"periodType":      value,
				"periodInterval":  "1",
				"trialPeriodType": value,
			})
			if err != nil {
				t.Fatalf("marshal product info: %v", err)
			}
			var product subscription.ProductInfo
			if err := json.Unmarshal(productJSON, &product); err != nil {
				t.Fatalf("parse subscription.ProductInfo: %v", err)
			}
			if product.PeriodType != value || product.TrialPeriodType != value {
				t.Errorf("subscription.ProductInfo period types: got %q/%q want %q",
					product.PeriodType, product.TrialPeriodType, value)
			}
			productValueJSON, err := json.Marshal(product)
			if err != nil {
				t.Fatalf("serialize subscription.ProductInfo: %v", err)
			}
			var productFields map[string]interface{}
			if err := json.Unmarshal(productValueJSON, &productFields); err != nil {
				t.Fatalf("decode serialized subscription.ProductInfo: %v", err)
			}
			if productFields["periodType"] != value || productFields["trialPeriodType"] != value {
				t.Errorf("serialized subscription.ProductInfo period types: got %#v/%#v want %q",
					productFields["periodType"], productFields["trialPeriodType"], value)
			}
		})
	}

	var missing subscription.PaymentDetail
	if err := json.Unmarshal([]byte(`{}`), &missing); err != nil {
		t.Fatalf("parse subscription.PaymentDetail without orderStatus: %v", err)
	}
	if missing.OrderStatus != "" {
		t.Errorf("missing orderStatus must keep the Go zero value, got %q", missing.OrderStatus)
	}

	var nullStatus subscription.PaymentDetail
	if err := json.Unmarshal([]byte(`{"orderStatus":null}`), &nullStatus); err != nil {
		t.Fatalf("parse subscription.PaymentDetail with null orderStatus: %v", err)
	}
	if nullStatus.OrderStatus != "" {
		t.Errorf("null orderStatus must keep the Go zero value, got %q", nullStatus.OrderStatus)
	}
	nullJSON, err := json.Marshal(nullStatus)
	if err != nil {
		t.Fatalf("serialize subscription.PaymentDetail with null orderStatus: %v", err)
	}
	var nullFields map[string]interface{}
	if err := json.Unmarshal(nullJSON, &nullFields); err != nil {
		t.Fatalf("decode serialized subscription.PaymentDetail with null orderStatus: %v", err)
	}
	if _, present := nullFields["orderStatus"]; present {
		t.Errorf("zero orderStatus must stay omitted by the existing omitempty tag, got %#v", nullFields)
	}
}

func TestOpenValuesBuiltinFailurePayloads(t *testing.T) {
	vector := loadOpenValuesVector(t)
	if len(vector.FailureEvents) != 4 {
		t.Fatal("missing failure payloads")
	}
	for _, event := range vector.FailureEvents {
		t.Run(event.EventType, func(t *testing.T) {
			handler, keys := newOpenValuesWebhookHandler(t)
			var received map[string]interface{}
			record := func(value interface{}) {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &received); err != nil {
					t.Fatal(err)
				}
			}
			handler.OnPayment(func(v *core.PaymentNotification) { record(v) }).
				OnRefund(func(v *core.RefundNotification) { record(v) }).
				OnSubscriptionStatus(func(v *core.SubscriptionStatusNotification) { record(v) }).
				OnSubscriptionChange(func(v *core.SubscriptionChangeNotification) { record(v) })
			body := openValuesKnownEventBody(t, event)
			result := handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys))
			requireOpenValuesSuccess(t, result, keys, event.EventType)
			var expected map[string]interface{}
			if err := json.Unmarshal([]byte(body), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, received) {
				t.Fatalf("lost failure fields: expected=%v actual=%v", expected, received)
			}
		})
	}
}

func TestOpenValuesPanicNilProducesFailure(t *testing.T) {
	t.Setenv("GODEBUG", "panicnil=1")
	vector := loadOpenValuesVector(t)
	handler, keys := newOpenValuesWebhookHandler(t)
	_, err := handler.OnEvent(vector.CustomEvent.EventType, func(json.RawMessage) error { panic(nil) })
	if err != nil {
		t.Fatal(err)
	}
	body := openValuesCustomEventBody(t, vector.CustomEvent)
	requireOpenValuesFailure(t, handler.HandleWebhook(body, signOpenValuesPayload(t, body, keys)), keys, "panic(nil)")
}

package core

import (
	"encoding/json"
	"fmt"

	"github.com/waffo-com/waffo-go/config"
	"github.com/waffo-com/waffo-go/types/subscription"
	"github.com/waffo-com/waffo-go/utils"
)

// Webhook event types
const (
	EventPayment                   = "PAYMENT_NOTIFICATION"
	EventRefund                    = "REFUND_NOTIFICATION"
	EventSubscriptionStatus        = "SUBSCRIPTION_STATUS_NOTIFICATION"
	EventSubscriptionPeriodChanged = "SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION"
	EventSubscriptionChange        = "SUBSCRIPTION_CHANGE_NOTIFICATION"
)

// WebhookResult represents the result of webhook processing.
type WebhookResult struct {
	Success           bool
	ResponseBody      string
	ResponseSignature string
	Error             string
}

// WebhookEvent represents the base webhook event.
type WebhookEvent struct {
	EventType string `json:"eventType"`
}

// PaymentNotificationResult contains the result data of a payment notification.
type PaymentNotificationResult struct {
	PaymentRequestID string                 `json:"paymentRequestId,omitempty"`
	MerchantOrderID  string                 `json:"merchantOrderId,omitempty"`
	AcquiringOrderID string                 `json:"acquiringOrderId,omitempty"`
	OrderStatus      string                 `json:"orderStatus,omitempty"`
	OrderAction      string                 `json:"orderAction,omitempty"`
	OrderCurrency    string                 `json:"orderCurrency,omitempty"`
	OrderAmount      string                 `json:"orderAmount,omitempty"`
	UserCurrency     string                 `json:"userCurrency,omitempty"`
	FinalDealAmount  string                 `json:"finalDealAmount,omitempty"`
	OrderDescription string                 `json:"orderDescription,omitempty"`
	MerchantInfo     map[string]interface{} `json:"merchantInfo,omitempty"`
	UserInfo         map[string]interface{} `json:"userInfo,omitempty"`
	GoodsInfo        map[string]interface{} `json:"goodsInfo,omitempty"`
	AddressInfo      map[string]interface{} `json:"addressInfo,omitempty"`
	PaymentInfo      map[string]interface{} `json:"paymentInfo,omitempty"`
	OrderRequestedAt string                 `json:"orderRequestedAt,omitempty"`
	OrderExpiredAt   string                 `json:"orderExpiredAt,omitempty"`
	OrderUpdatedAt   string                 `json:"orderUpdatedAt,omitempty"`
	OrderCompletedAt string                 `json:"orderCompletedAt,omitempty"`
	OrderFailedReason  map[string]interface{}       `json:"orderFailedReason,omitempty"`
	ExtendInfo         string                       `json:"extendInfo,omitempty"`
	SubscriptionInfo   *subscription.SubscriptionInfo `json:"subscriptionInfo,omitempty"`
	RefundExpiryAt     string                       `json:"refundExpiryAt,omitempty"`
	CancelRedirectUrl  string                       `json:"cancelRedirectUrl,omitempty"`
}

// PaymentNotification represents a payment webhook notification.
type PaymentNotification struct {
	EventType string                     `json:"eventType"`
	Result    *PaymentNotificationResult `json:"result,omitempty"`
}

// RefundNotificationResult contains the result data of a refund notification.
type RefundNotificationResult struct {
	RefundRequestID        string                 `json:"refundRequestId,omitempty"`
	MerchantRefundOrderID  string                 `json:"merchantRefundOrderId,omitempty"`
	AcquiringOrderID       string                 `json:"acquiringOrderId,omitempty"`
	AcquiringRefundOrderID string                 `json:"acquiringRefundOrderId,omitempty"`
	OrigPaymentRequestID   string                 `json:"origPaymentRequestId,omitempty"`
	RefundAmount           string                 `json:"refundAmount,omitempty"`
	RefundStatus           string                 `json:"refundStatus,omitempty"`
	RemainingRefundAmount  string                 `json:"remainingRefundAmount,omitempty"`
	UserCurrency           string                 `json:"userCurrency,omitempty"`
	FinalDealAmount        string                 `json:"finalDealAmount,omitempty"`
	RefundReason           string                 `json:"refundReason,omitempty"`
	RefundRequestedAt      string                 `json:"refundRequestedAt,omitempty"`
	RefundUpdatedAt        string                 `json:"refundUpdatedAt,omitempty"`
	RefundCompletedAt      string                 `json:"refundCompletedAt,omitempty"`
	RefundFailedReason     map[string]interface{} `json:"refundFailedReason,omitempty"`
	UserInfo               map[string]interface{} `json:"userInfo,omitempty"`
	RefundSource           string                 `json:"refundSource,omitempty"`
	ExtendInfo             string                 `json:"extendInfo,omitempty"`
}

// RefundNotification represents a refund webhook notification.
type RefundNotification struct {
	EventType string                    `json:"eventType"`
	Result    *RefundNotificationResult `json:"result,omitempty"`
}

// SubscriptionNotificationResult contains the result data of a subscription notification.
// Shared by SubscriptionStatusNotification and SubscriptionPeriodChangedNotification.
type SubscriptionNotificationResult struct {
	SubscriptionRequest       string                   `json:"subscriptionRequest,omitempty"`
	MerchantSubscriptionID    string                   `json:"merchantSubscriptionId,omitempty"`
	SubscriptionID            string                   `json:"subscriptionId,omitempty"`
	PayMethodSubscriptionID   string                   `json:"payMethodSubscriptionId,omitempty"`
	SubscriptionStatus        string                   `json:"subscriptionStatus,omitempty"`
	SubscriptionAction        string                   `json:"subscriptionAction,omitempty"`
	Currency                  string                   `json:"currency,omitempty"`
	Amount                    string                   `json:"amount,omitempty"`
	UserCurrency              string                   `json:"userCurrency,omitempty"`
	ProductInfo               map[string]interface{}   `json:"productInfo,omitempty"`
	MerchantInfo              map[string]interface{}   `json:"merchantInfo,omitempty"`
	UserInfo                  map[string]interface{}   `json:"userInfo,omitempty"`
	GoodsInfo                 map[string]interface{}   `json:"goodsInfo,omitempty"`
	AddressInfo               map[string]interface{}   `json:"addressInfo,omitempty"`
	PaymentInfo               map[string]interface{}   `json:"paymentInfo,omitempty"`
	RequestedAt               string                   `json:"requestedAt,omitempty"`
	UpdatedAt                 string                   `json:"updatedAt,omitempty"`
	FailedReason              map[string]interface{}   `json:"failedReason,omitempty"`
	ExtendInfo                string                   `json:"extendInfo,omitempty"`
	SubscriptionManagementURL string                   `json:"subscriptionManagementUrl,omitempty"`
	PaymentDetails            []map[string]interface{} `json:"paymentDetails,omitempty"`
}

// SubscriptionStatusNotification represents a subscription status webhook notification.
type SubscriptionStatusNotification struct {
	EventType string                          `json:"eventType"`
	Result    *SubscriptionNotificationResult `json:"result,omitempty"`
}

// SubscriptionPeriodChangedNotification represents a subscription period changed webhook notification.
type SubscriptionPeriodChangedNotification struct {
	EventType string                          `json:"eventType"`
	Result    *SubscriptionNotificationResult `json:"result,omitempty"`
}

// SubscriptionChangeNotificationResult contains the result data of a subscription change notification.
// Triggered when a subscription change (upgrade/downgrade) reaches final status (SUCCESS or CLOSED).
type SubscriptionChangeNotificationResult struct {
	SubscriptionRequest       string                   `json:"subscriptionRequest,omitempty"`
	OriginSubscriptionRequest string                   `json:"originSubscriptionRequest,omitempty"`
	MerchantSubscriptionID    string                   `json:"merchantSubscriptionId,omitempty"`
	SubscriptionID            string                   `json:"subscriptionId,omitempty"`
	SubscriptionChangeStatus  string                   `json:"subscriptionChangeStatus,omitempty"`
	SubscriptionAction        string                   `json:"subscriptionAction,omitempty"`
	RemainingAmount           string                   `json:"remainingAmount,omitempty"`
	Currency                  string                   `json:"currency,omitempty"`
	UserCurrency              string                   `json:"userCurrency,omitempty"`
	RequestedAt               string                   `json:"requestedAt,omitempty"`
	SubscriptionManagementURL string                   `json:"subscriptionManagementUrl,omitempty"`
	ExtendInfo                string                   `json:"extendInfo,omitempty"`
	OrderExpiredAt            string                   `json:"orderExpiredAt,omitempty"`
	ProductInfoList           []map[string]interface{} `json:"productInfoList,omitempty"`
	MerchantInfo              map[string]interface{}   `json:"merchantInfo,omitempty"`
	UserInfo                  map[string]interface{}   `json:"userInfo,omitempty"`
	GoodsInfo                 map[string]interface{}   `json:"goodsInfo,omitempty"`
	AddressInfo               map[string]interface{}   `json:"addressInfo,omitempty"`
	PaymentInfo               map[string]interface{}   `json:"paymentInfo,omitempty"`
}

// SubscriptionChangeNotification represents a subscription change webhook notification.
type SubscriptionChangeNotification struct {
	EventType string                                `json:"eventType"`
	Result    *SubscriptionChangeNotificationResult `json:"result,omitempty"`
}

// Handler function types
type (
	PaymentHandler                   func(*PaymentNotification)
	RefundHandler                    func(*RefundNotification)
	SubscriptionStatusHandler        func(*SubscriptionStatusNotification)
	SubscriptionPaymentHandler       func(*SubscriptionStatusNotification)
	SubscriptionPeriodChangedHandler func(*SubscriptionPeriodChangedNotification)
	SubscriptionChangeHandler        func(*SubscriptionChangeNotification)
)

// WebhookHandler handles incoming webhook notifications.
type WebhookHandler struct {
	config                           *config.WaffoConfig
	paymentHandler                   PaymentHandler
	refundHandler                    RefundHandler
	subscriptionStatusHandler        SubscriptionStatusHandler
	subscriptionPaymentHandler       SubscriptionPaymentHandler
	subscriptionPeriodChangedHandler SubscriptionPeriodChangedHandler
	subscriptionChangeHandler        SubscriptionChangeHandler
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(cfg *config.WaffoConfig) *WebhookHandler {
	return &WebhookHandler{
		config: cfg,
	}
}

// OnPayment registers a handler for payment notifications.
func (h *WebhookHandler) OnPayment(handler PaymentHandler) *WebhookHandler {
	h.paymentHandler = handler
	return h
}

// OnRefund registers a handler for refund notifications.
func (h *WebhookHandler) OnRefund(handler RefundHandler) *WebhookHandler {
	h.refundHandler = handler
	return h
}

// OnSubscriptionStatus registers a handler for subscription status notifications.
func (h *WebhookHandler) OnSubscriptionStatus(handler SubscriptionStatusHandler) *WebhookHandler {
	h.subscriptionStatusHandler = handler
	return h
}

// OnSubscriptionPayment registers a handler for subscription payment notifications.
// Note: Subscription payment notifications arrive as SUBSCRIPTION_STATUS_NOTIFICATION events.
// If both OnSubscriptionStatus and OnSubscriptionPayment are registered, OnSubscriptionStatus takes priority.
// OnSubscriptionPayment is used as a fallback when OnSubscriptionStatus is not registered.
func (h *WebhookHandler) OnSubscriptionPayment(handler SubscriptionPaymentHandler) *WebhookHandler {
	h.subscriptionPaymentHandler = handler
	return h
}

// OnSubscriptionPeriodChanged registers a handler for subscription period changed notifications.
func (h *WebhookHandler) OnSubscriptionPeriodChanged(handler SubscriptionPeriodChangedHandler) *WebhookHandler {
	h.subscriptionPeriodChangedHandler = handler
	return h
}

// OnSubscriptionChange registers a handler for subscription change notifications.
// Triggered when a subscription change (upgrade/downgrade) reaches final status (SUCCESS or CLOSED).
func (h *WebhookHandler) OnSubscriptionChange(handler SubscriptionChangeHandler) *WebhookHandler {
	h.subscriptionChangeHandler = handler
	return h
}

// HandleWebhook processes an incoming webhook request.
func (h *WebhookHandler) HandleWebhook(body string, signature string) *WebhookResult {
	// 1. Verify signature
	if signature == "" {
		return h.buildFailedResult("missing signature")
	}

	if !h.VerifySignature(body, signature) {
		return h.buildFailedResult("invalid signature")
	}

	// 2. Parse event type
	var event WebhookEvent
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return h.buildFailedResult("failed to parse webhook body")
	}

	// 3. Route to handler
	var handlerErr error
	switch event.EventType {
	case EventPayment:
		handlerErr = h.handlePayment(body)
	case EventRefund:
		handlerErr = h.handleRefund(body)
	case EventSubscriptionStatus:
		// Priority: subscriptionStatusHandler > subscriptionPaymentHandler (fallback)
		if h.subscriptionStatusHandler != nil {
			handlerErr = h.handleSubscriptionStatus(body)
		} else if h.subscriptionPaymentHandler != nil {
			handlerErr = h.handleSubscriptionPayment(body)
		}
	case EventSubscriptionPeriodChanged:
		handlerErr = h.handleSubscriptionPeriodChanged(body)
	case EventSubscriptionChange:
		handlerErr = h.handleSubscriptionChange(body)
	default:
		return h.buildFailedResult(fmt.Sprintf("unknown event type: %s", event.EventType))
	}

	if handlerErr != nil {
		return h.buildFailedResult(handlerErr.Error())
	}

	// 4. Build success response
	return h.buildSuccessResult()
}

// VerifySignature verifies the webhook signature.
func (h *WebhookHandler) VerifySignature(body string, signature string) bool {
	return utils.Verify(body, signature, h.config.WaffoPublicKey)
}

// BuildSuccessResponse builds a signed success response.
func (h *WebhookHandler) BuildSuccessResponse() (body string, signature string) {
	resp := map[string]string{"message": "success"}
	bodyBytes, _ := json.Marshal(resp)
	body = string(bodyBytes)
	signature, _ = utils.Sign(body, h.config.PrivateKey)
	return
}

// BuildFailedResponse builds a signed failed response.
func (h *WebhookHandler) BuildFailedResponse(message string) (body string, signature string) {
	resp := map[string]string{"message": "failed"}
	bodyBytes, _ := json.Marshal(resp)
	body = string(bodyBytes)
	signature, _ = utils.Sign(body, h.config.PrivateKey)
	return
}

func (h *WebhookHandler) buildSuccessResult() *WebhookResult {
	body, sig := h.BuildSuccessResponse()
	return &WebhookResult{
		Success:           true,
		ResponseBody:      body,
		ResponseSignature: sig,
	}
}

func (h *WebhookHandler) buildFailedResult(message string) *WebhookResult {
	body, sig := h.BuildFailedResponse(message)
	return &WebhookResult{
		Success:           false,
		ResponseBody:      body,
		ResponseSignature: sig,
		Error:             message,
	}
}

func (h *WebhookHandler) handlePayment(body string) error {
	if h.paymentHandler == nil {
		return nil // No handler registered, just acknowledge
	}

	var notification PaymentNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		return fmt.Errorf("failed to parse payment notification: %w", err)
	}

	// Call handler (catch panics to prevent webhook failure)
	defer func() {
		if r := recover(); r != nil {
			_ = r // Intentionally suppress panic, webhook should not fail
		}
	}()

	h.paymentHandler(&notification)
	return nil
}

func (h *WebhookHandler) handleRefund(body string) error {
	if h.refundHandler == nil {
		return nil
	}

	var notification RefundNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		return fmt.Errorf("failed to parse refund notification: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = r // Intentionally suppress panic
		}
	}()

	h.refundHandler(&notification)
	return nil
}

func (h *WebhookHandler) handleSubscriptionStatus(body string) error {
	if h.subscriptionStatusHandler == nil {
		return nil
	}

	var notification SubscriptionStatusNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		return fmt.Errorf("failed to parse subscription status notification: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = r // Intentionally suppress panic
		}
	}()

	h.subscriptionStatusHandler(&notification)
	return nil
}

func (h *WebhookHandler) handleSubscriptionPayment(body string) error {
	if h.subscriptionPaymentHandler == nil {
		return nil
	}

	var notification SubscriptionStatusNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		return fmt.Errorf("failed to parse subscription payment notification: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = r // Intentionally suppress panic
		}
	}()

	h.subscriptionPaymentHandler(&notification)
	return nil
}

func (h *WebhookHandler) handleSubscriptionPeriodChanged(body string) error {
	if h.subscriptionPeriodChangedHandler == nil {
		return nil
	}

	var notification SubscriptionPeriodChangedNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		return fmt.Errorf("failed to parse subscription period changed notification: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = r // Intentionally suppress panic
		}
	}()

	h.subscriptionPeriodChangedHandler(&notification)
	return nil
}

func (h *WebhookHandler) handleSubscriptionChange(body string) error {
	if h.subscriptionChangeHandler == nil {
		return nil
	}

	var notification SubscriptionChangeNotification
	if err := json.Unmarshal([]byte(body), &notification); err != nil {
		return fmt.Errorf("failed to parse subscription change notification: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = r // Intentionally suppress panic
		}
	}()

	h.subscriptionChangeHandler(&notification)
	return nil
}

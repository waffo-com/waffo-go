# Waffo Go SDK

<!-- Synced with waffo-sdk/README.md @ commit 1160423 -->

<!-- Synced with waffo-sdk/README.md -->

**English** | [中文](README_CN.md)

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official Go SDK for [Waffo Payment Platform](https://www.waffo.com), providing one-stop global payment solutions for AI products, SaaS services, and more.

## Introduction

### Core Features

- **Global Payments**: Support for credit cards, debit cards, e-wallets, virtual accounts, and more payment methods covering mainstream global payment channels
- **Subscription Management**: Complete subscription lifecycle management with trial periods, recurring billing, and subscription upgrades/downgrades
- **Refund Processing**: Flexible full/partial refund capabilities with refund status tracking
- **Webhook Notifications**: Real-time payment result push notifications for payments, refunds, subscription status changes, and more
- **Security & Reliability**: PCI DSS certified, RSA signature verification, enforced TLS 1.2+ encryption

### Use Cases

| Scenario | Description |
|----------|-------------|
| **AI Products** | ChatGPT-like applications, AI writing tools, AI image generation with usage-based billing or subscriptions |
| **SaaS Services** | Enterprise software subscriptions, online collaboration tools, cloud services with periodic payments |
| **Content Platforms** | Membership subscriptions, paid content, tipping scenarios |

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Framework Integration](#framework-integration)
- [API Usage](#api-usage)
  - [Order Management](#order-management)
  - [Subscription Management](#subscription-management)
  - [Refund Query](#refund-query)
  - [Merchant Configuration](#merchant-configuration)
- [Webhook Handling](#webhook-handling)
- [Advanced Configuration](#advanced-configuration)
  - [Custom HTTP Transport](#custom-http-transport)
  - [TLS Security Configuration](#tls-security-configuration)
- [Error Handling](#error-handling)
- [Testing](#testing)
  - [Unit Tests](#unit-tests)
  - [E2E Tests](#e2e-tests)
- [Support](#support)
- [License](#license)
- [Payment Method Types](#payment-method-types)
- [Handling New API Fields (ExtraParams)](#handling-new-api-fields-extraparams)

## Requirements

- Go 1.20+
- No external dependencies (uses standard library only)

### Version Compatibility

| Go Version | Support Status |
|------------|----------------|
| 1.22.x | ✅ Fully Supported |
| 1.21.x | ✅ Fully Supported (Recommended) |
| 1.20.x | ✅ Fully Supported |
| < 1.20 | ❌ Not Supported |

## Installation

```bash
go get github.com/waffo-com/waffo-go
```

## Quick Start

### 1. Initialize the SDK

```go
package main

import (
    "github.com/waffo-com/waffo-go"
    "github.com/waffo-com/waffo-go/config"
)

func main() {
    cfg, err := config.NewConfigBuilder().
        APIKey("your-api-key").
        PrivateKey("your-base64-encoded-private-key").
        WaffoPublicKey("waffo-public-key").           // From Waffo Dashboard
        MerchantID("your-merchant-id").               // Auto-injected into requests
        Environment(config.Sandbox).                   // Sandbox or Production
        Build()
    if err != nil {
        log.Fatal(err)
    }

    client := waffo.New(cfg)
}
```

### 2. Create a Payment Order

```go
import (
    "context"
    "github.com/google/uuid"
    "github.com/waffo-com/waffo-go/types/order"
)

ctx := context.Background()

// Generate idempotency key (max 32 chars)
paymentRequestID := uuid.New().String()[:32]

resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    MerchantOrderID:  "ORDER-" + uuid.New().String()[:8],
    PaymentRequestID: paymentRequestID,
    OrderAmount:      "99.99",
    OrderCurrency:    "USD",
    Subject:          "Premium Subscription",
    PaymentMethod:    "CARD",
    // ... other parameters
}, nil)

if err != nil {
    log.Fatal(err)
}

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Payment URL: %s\n", data.PaymentURL)
} else {
    fmt.Printf("Error: %s - %s\n", resp.ResultCode, resp.ResultMsg)
}
```

### 3. Handle Webhook Notifications

```go
import (
    "github.com/waffo-com/waffo-go/core"
)

handler := client.Webhook().
    OnPayment(func(n *core.PaymentNotification) {
        fmt.Printf("Payment %s: %s\n", n.PaymentRequestID, n.OrderStatus)
    }).
    OnRefund(func(n *core.RefundNotification) {
        fmt.Printf("Refund %s: %s\n", n.RefundRequestID, n.RefundStatus)
    })

// In your HTTP handler
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Waffo-Signature")

    result := handler.HandleWebhook(string(body), signature)

    if result.Success {
        responseBody, responseSig := handler.BuildSuccessResponse()
        w.Header().Set("X-Waffo-Signature", responseSig)
        w.Write([]byte(responseBody))
    } else {
        responseBody, responseSig := handler.BuildFailedResponse(result.Error)
        w.Header().Set("X-Waffo-Signature", responseSig)
        w.WriteHeader(http.StatusBadRequest)
        w.Write([]byte(responseBody))
    }
}
```

## Configuration

### Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `APIKey` | `string` | ✅ | - | API key from Waffo Dashboard |
| `PrivateKey` | `string` | ✅ | - | Base64-encoded PKCS#8 private key |
| `WaffoPublicKey` | `string` | ✅ | - | Base64-encoded X.509 public key from Waffo |
| `MerchantID` | `string` | ❌ | - | Merchant ID for auto-injection |
| `Environment` | `Environment` | ❌ | `Sandbox` | `Sandbox` or `Production` |
| `Timeout` | `time.Duration` | ❌ | `30s` | Request timeout |

### Environment Variables

```go
// Initialize from environment variables
client, err := waffo.FromEnv()
```

| Variable | Description |
|----------|-------------|
| `WAFFO_API_KEY` | API key (required) |
| `WAFFO_PRIVATE_KEY` | Private key in Base64 (required) |
| `WAFFO_PUBLIC_KEY` | Waffo's public key in Base64 (required) |
| `WAFFO_ENVIRONMENT` | `SANDBOX` or `PRODUCTION` (default: SANDBOX) |
| `WAFFO_MERCHANT_ID` | Merchant ID (optional) |

### Framework Integration

#### Gin

```go
import (
    "github.com/gin-gonic/gin"
)

func setupRoutes(r *gin.Engine, waffoClient *waffo.Waffo) {
    handler := waffoClient.Webhook().
        OnPayment(func(n *core.PaymentNotification) {
            // Process payment notification
        })

    r.POST("/webhook/waffo", func(c *gin.Context) {
        body, _ := io.ReadAll(c.Request.Body)
        signature := c.GetHeader("X-Waffo-Signature")

        result := handler.HandleWebhook(string(body), signature)

        if result.Success {
            responseBody, responseSig := handler.BuildSuccessResponse()
            c.Header("X-Waffo-Signature", responseSig)
            c.String(http.StatusOK, responseBody)
        } else {
            responseBody, responseSig := handler.BuildFailedResponse(result.Error)
            c.Header("X-Waffo-Signature", responseSig)
            c.String(http.StatusBadRequest, responseBody)
        }
    })
}
```

#### Echo

```go
import (
    "github.com/labstack/echo/v4"
)

func setupRoutes(e *echo.Echo, waffoClient *waffo.Waffo) {
    handler := waffoClient.Webhook().
        OnPayment(func(n *core.PaymentNotification) {
            // Process payment notification
        })

    e.POST("/webhook/waffo", func(c echo.Context) error {
        body, _ := io.ReadAll(c.Request().Body)
        signature := c.Request().Header.Get("X-Waffo-Signature")

        result := handler.HandleWebhook(string(body), signature)

        if result.Success {
            responseBody, responseSig := handler.BuildSuccessResponse()
            c.Response().Header().Set("X-Waffo-Signature", responseSig)
            return c.String(http.StatusOK, responseBody)
        }
        responseBody, responseSig := handler.BuildFailedResponse(result.Error)
        c.Response().Header().Set("X-Waffo-Signature", responseSig)
        return c.String(http.StatusBadRequest, responseBody)
    })
}
```

#### Fiber

```go
import (
    "github.com/gofiber/fiber/v2"
)

func setupRoutes(app *fiber.App, waffoClient *waffo.Waffo) {
    handler := waffoClient.Webhook().
        OnPayment(func(n *core.PaymentNotification) {
            // Process payment notification
        })

    app.Post("/webhook/waffo", func(c *fiber.Ctx) error {
        body := c.Body()
        signature := c.Get("X-Waffo-Signature")

        result := handler.HandleWebhook(string(body), signature)

        if result.Success {
            responseBody, responseSig := handler.BuildSuccessResponse()
            c.Set("X-Waffo-Signature", responseSig)
            return c.SendString(responseBody)
        }
        responseBody, responseSig := handler.BuildFailedResponse(result.Error)
        c.Set("X-Waffo-Signature", responseSig)
        return c.Status(fiber.StatusBadRequest).SendString(responseBody)
    })
}
```

## API Usage

### Order Management

#### Create Order

```go
resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    MerchantOrderID:  "ORDER-123",
    PaymentRequestID: "REQ-123",
    OrderAmount:      "99.99",
    OrderCurrency:    "USD",
    Subject:          "Premium Plan",
    PaymentMethod:    "CARD",
    ReturnURL:        "https://example.com/return",
    NotifyURL:        "https://example.com/webhook",
}, nil)
```

#### Query Order

```go
resp, err := client.Order().Query(ctx, &order.QueryOrderParams{
    PaymentRequestID: "REQ-123",
}, nil)

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Order Status: %s\n", data.OrderStatus)
}
```

#### Close Order

```go
resp, err := client.Order().Close(ctx, &order.CloseOrderParams{
    PaymentRequestID: "REQ-123",
}, nil)
```

### Subscription Management

#### Create Subscription

```go
resp, err := client.Subscription().Create(ctx, &subscription.CreateSubscriptionParams{
    SubscriptionRequest:   "SUB-REQ-123",
    SubscriptionPlanID:    "PLAN-001",
    PaymentMethod:         "CARD",
    OrderAmount:           "9.99",
    OrderCurrency:         "USD",
    BillingCycle:          "MONTH",
    BillingCycleCount:     1,
    ReturnURL:             "https://example.com/return",
    NotifyURL:             "https://example.com/webhook",
}, nil)
```

#### Query Subscription

```go
resp, err := client.Subscription().Query(ctx, &subscription.QuerySubscriptionParams{
    SubscriptionRequest: "SUB-REQ-123",
}, nil)
```

#### Cancel Subscription

```go
resp, err := client.Subscription().Cancel(ctx, &subscription.CancelSubscriptionParams{
    SubscriptionRequest: "SUB-REQ-123",
}, nil)
```

### Subscription Change (Upgrade/Downgrade)

Change an existing subscription to a new plan (upgrade or downgrade).

#### Change Subscription

```go
// New subscription request ID for the change
subscriptionRequest := fmt.Sprintf("%x", time.Now().UnixNano())[:32]
originSubscriptionRequest := "original-subscription-request-id"

resp, err := client.Subscription().Change(ctx, &subscription.ChangeSubscriptionParams{
    SubscriptionRequest:       subscriptionRequest,
    OriginSubscriptionRequest: originSubscriptionRequest,
    RemainingAmount:           "50.00", // Remaining value from original subscription
    Currency:                  "HKD",
    RequestedAt:               time.Now().UTC().Format(time.RFC3339),
    NotifyURL:                 "https://your-site.com/webhook/subscription",
    ProductInfoList: []subscription.SubscriptionChangeProductInfo{
        {
            Description:    "Yearly Premium Subscription",
            PeriodType:     "YEAR",
            PeriodInterval: "1",
            Amount:         "999.00",
        },
    },
    UserInfo: &subscription.SubscriptionUserInfo{
        UserID:    "user_123",
        UserEmail: "user@example.com",
    },
    GoodsInfo: &subscription.SubscriptionGoodsInfo{
        GoodsID:   "GOODS_PREMIUM",
        GoodsName: "Premium Plan",
    },
    PaymentInfo: &subscription.SubscriptionPaymentInfo{
        ProductName: "SUBSCRIPTION",
    },
    MerchantInfo: &subscription.SubscriptionMerchantInfo{},
    // Optional fields
    MerchantSubscriptionID:    fmt.Sprintf("MSUB_UPGRADE_%d", time.Now().UnixMilli()),
    SuccessRedirectURL:        "https://your-site.com/subscription/upgrade/success",
    FailedRedirectURL:         "https://your-site.com/subscription/upgrade/failed",
    CancelRedirectURL:         "https://your-site.com/subscription/upgrade/cancel",
    SubscriptionManagementURL: "https://your-site.com/subscription/manage",
}, nil)

if err != nil {
    var unknownErr *errors.WaffoUnknownStatusError
    if errors.As(err, &unknownErr) {
        // Status unknown - DO NOT assume failure! User may have completed payment
        log.Printf("Unknown status, need to query: %v", unknownErr)

        // Correct handling: Call inquiry API to confirm actual status
        inquiryResp, _ := client.Subscription().ChangeInquiry(ctx, &subscription.ChangeInquiryParams{
            SubscriptionRequest: subscriptionRequest,
        }, nil)
        _ = inquiryResp
        // Or wait for Webhook callback
    } else {
        log.Fatal(err)
    }
} else if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Change Status: %s\n", data.SubscriptionChangeStatus)
    fmt.Printf("New Subscription ID: %s\n", data.SubscriptionID)

    if data.SubscriptionChangeStatus == "AUTHORIZATION_REQUIRED" {
        // User needs to authorize the change
        fmt.Printf("Redirect user to: %s\n", data.FetchRedirectURL())
    } else if data.SubscriptionChangeStatus == "SUCCESS" {
        // Change completed successfully
        fmt.Println("Subscription upgraded successfully")
    }
}
```

#### Subscription Change Status Values

| Status | Description |
|--------|-------------|
| `IN_PROGRESS` | Change is being processed |
| `AUTHORIZATION_REQUIRED` | User needs to authorize the change (redirect to webUrl) |
| `SUCCESS` | Change completed successfully |
| `CLOSED` | Change was closed (timeout or failed) |

#### Query Subscription Change Status

```go
resp, err := client.Subscription().ChangeInquiry(ctx, &subscription.ChangeInquiryParams{
    SubscriptionRequest: "new-subscription-request-id",
}, nil)
if err != nil {
    log.Fatal(err)
}
if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Change Status: %s\n", data.SubscriptionChangeStatus)
    fmt.Printf("New Subscription ID: %s\n", data.SubscriptionID)
}
```

### Refund Query

```go
resp, err := client.Refund().Query(ctx, &refund.QueryRefundParams{
    RefundRequestID: "REFUND-123",
}, nil)

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Refund Status: %s\n", data.RefundStatus)
}
```

### Merchant Configuration

```go
resp, err := client.MerchantConfig().Inquiry(ctx, &merchant.InquiryMerchantConfigParams{
    // params
}, nil)
```

## Webhook Handling

### Handler Chaining

```go
handler := client.Webhook().
    OnPayment(func(n *core.PaymentNotification) {
        // Handle payment
    }).
    OnRefund(func(n *core.RefundNotification) {
        // Handle refund
    }).
    OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {
        // Handle subscription status
    }).
    OnSubscriptionPayment(func(n *core.SubscriptionPaymentNotification) {
        // Handle subscription payment
    }).
    OnSubscriptionPeriodChanged(func(n *core.SubscriptionPeriodChangedNotification) {
        // Handle period change
    })
```

### Webhook Notification Types

| Event Type | Handler Method | Description |
|------------|----------------|-------------|
| `PAYMENT_NOTIFICATION` | `onPayment()` | Payment result notification (triggered on every payment attempt, including retries) |
| `REFUND_NOTIFICATION` | `onRefund()` | Refund result notification |
| `SUBSCRIPTION_STATUS_NOTIFICATION` | `onSubscriptionStatus()` | Subscription status change notification (triggered when subscription main record status changes) |
| `SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION` | `onSubscriptionPeriodChanged()` | Subscription period change notification (final result of each period) |
| `SUBSCRIPTION_CHANGE_NOTIFICATION` | `onSubscriptionChange()` | Subscription change (upgrade/downgrade) result notification |

### Subscription Notification Types Explained

| Notification Type | Trigger Condition | Scope | Includes Retry Events | Typical Use Case |
|-------------------|-------------------|-------|----------------------|------------------|
| `SUBSCRIPTION_STATUS_NOTIFICATION` | Subscription main record status changes | Subscription level | No | Track subscription lifecycle: first payment success activation (ACTIVE), cancellation (MERCHANT_CANCELLED, CHANNEL_CANCELLED), first payment failure close (CLOSE), etc. |
| `SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION` | Subscription period reaches final state | Period level | No (only final result) | Only need final result of each period, no intermediate retry events |
| `SUBSCRIPTION_CHANGE_NOTIFICATION` | Subscription change (upgrade/downgrade) completes | Change request level | No (only final result) | Track subscription change results: SUCCESS or CLOSED |
| `PAYMENT_NOTIFICATION` | Every payment order | Payment order level | Yes (includes all retries) | Need complete details of every payment attempt, including failure reasons, timestamps, retry details |

> **Selection Guide**:
> - If you only care about subscription activation/cancellation, use `SUBSCRIPTION_STATUS_NOTIFICATION`
> - If you only care about final renewal result of each period, use `SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION`
> - If you only care about subscription change (upgrade/downgrade) final result, use `SUBSCRIPTION_CHANGE_NOTIFICATION`
> - If you need to track every payment attempt (including retries), use `PAYMENT_NOTIFICATION`

> **Subscription Payment Note**: Each period's payment (including first payment and renewals) triggers `PAYMENT_NOTIFICATION` events. You can get subscription-related info (subscriptionId, period, etc.) from `subscriptionInfo`.

> **Subscription Change (Upgrade/Downgrade) Webhook Note**:
> When a subscription change is processed, the following notifications are triggered:
> - `SUBSCRIPTION_CHANGE_NOTIFICATION`: When subscription change completes (SUCCESS or CLOSED)
> - `SUBSCRIPTION_STATUS_NOTIFICATION`: When original subscription status changes to `MERCHANT_CANCELLED`
> - `SUBSCRIPTION_STATUS_NOTIFICATION`: When new subscription status changes to `ACTIVE`
> - `PAYMENT_NOTIFICATION`: If upgrade requires additional payment (price difference)

### Webhook Notification Payload Examples

The following examples show the actual payload structure for each notification type.

#### PAYMENT_NOTIFICATION

```json
{
  "eventType": "PAYMENT_NOTIFICATION",
  "result": {
    "acquiringOrderId": "A2026xxxxxxxxxxxxxxxxxxxx",
    "orderStatus": "PAY_SUCCESS",
    "orderAmount": "109.00",
    "orderCurrency": "USD",
    "finalDealAmount": "109.00",
    "userCurrency": "USD",
    "orderDescription": "Sample Product",
    "orderRequestedAt": "2026-02-28T10:05:58.000Z",
    "orderCompletedAt": "2026-02-28T10:06:10.000Z",
    "orderUpdatedAt": "2026-02-28T10:06:10.000Z",
    "refundExpiryAt": "2026-08-26T23:59:59.999Z",
    "userInfo": {
      "userId": "user@example.com",
      "userEmail": "user@example.com"
    },
    "merchantInfo": {
      "merchantId": "YOUR_MERCHANT_ID"
    },
    "goodsInfo": {
      "goodsId": "goods_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "goodsName": "Sample Product"
    },
    "addressInfo": {},
    "paymentInfo": {
      "productName": "SUBSCRIPTION",
      "payMethodType": "CREDITCARD",
      "payMethodName": "CC_VISA",
      "payMethodProperties": "{\"cardToken\":\"CARD_TOKEN_XXXXXXXXXXXXXXXXXXXX\",\"cardTransactionType\":\"CIT\"}",
      "payMethodResponse": "{\"maskCardData\":\"XXXX42****XX4242\"}"
    },
    "subscriptionInfo": {
      "subscriptionId": "SC2026xxxxxxxxxxxxxxxxxxxxxxxxxx",
      "subscriptionRequest": "sub_req_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "merchantRequest": "sub_req_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "period": "1"
    },
    "cancelRedirectUrl": "https://YOUR_DOMAIN/checkout/YOUR_CHECKOUT_ID"
  }
}
```

#### SUBSCRIPTION_STATUS_NOTIFICATION / SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION

These two notification types share the same payload structure:

```json
{
  "eventType": "SUBSCRIPTION_STATUS_NOTIFICATION",
  "result": {
    "subscriptionId": "SC2026xxxxxxxxxxxxxxxxxxxxxxxxxx",
    "subscriptionRequest": "sub_req_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "merchantSubscriptionId": "sub_req_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "subscriptionStatus": "ACTIVE",
    "currency": "USD",
    "userCurrency": "USD",
    "amount": "109.00",
    "requestedAt": "2026-02-28T10:05:58.000Z",
    "updatedAt": "2026-02-28T10:06:10.000Z",
    "userInfo": {
      "userId": "user@example.com",
      "userEmail": "user@example.com"
    },
    "merchantInfo": {
      "merchantId": "YOUR_MERCHANT_ID"
    },
    "goodsInfo": {
      "goodsId": "goods_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "goodsName": "Sample Product"
    },
    "productInfo": {
      "periodType": "MONTHLY",
      "periodInterval": "1",
      "currentPeriod": "1",
      "startDateTime": "2026-02-28T10:06:10.000Z",
      "nextPaymentDateTime": "2026-03-28T10:06:10.000Z",
      "description": "Sample Product"
    },
    "paymentInfo": {
      "productName": "SUBSCRIPTION",
      "payMethodType": "CREDITCARD",
      "payMethodName": "CC_VISA",
      "payMethodProperties": "{}"
    },
    "paymentDetails": [
      {
        "period": "1",
        "acquiringOrderId": "A2026xxxxxxxxxxxxxxxxxxxx",
        "orderAmount": "109.00",
        "orderCurrency": "USD",
        "orderStatus": "PAY_SUCCESS",
        "orderUpdatedAt": "2026-02-28T10:06:10.000Z"
      }
    ]
  }
}
```

## Payment Method Types

### payMethodType Reference

| Type | Description | Example payMethodName |
|------|-------------|----------------------|
| `CREDITCARD` | Credit Card | CC_VISA, CC_MASTERCARD, CC_AMEX, CC_JCB, etc. |
| `DEBITCARD` | Debit Card | DC_VISA, DC_MASTERCARD, DC_ELO, etc. |
| `EWALLET` | E-Wallet | GCASH, DANA, PROMPTPAY, GRABPAY, etc. |
| `VA` | Virtual Account | BCA, BNI, BRI, MANDIRI, etc. |
| `APPLEPAY` | Apple Pay | APPLEPAY |
| `GOOGLEPAY` | Google Pay | GOOGLEPAY |

### Usage Examples

```typescript
// Specify type only, let user choose on checkout page
paymentInfo: {
  payMethodType: 'CREDITCARD',
}

// Specify exact payment method
paymentInfo: {
  payMethodType: 'CREDITCARD',
  payMethodName: 'CC_VISA',
}

// Combine multiple types
paymentInfo: {
  payMethodType: 'CREDITCARD,DEBITCARD',
}

// E-wallet with specific channel
paymentInfo: {
  payMethodType: 'EWALLET',
  payMethodName: 'GCASH',
}
```

> **Note**: For available `ProductName`, `PayMethodType`, `PayMethodName` values, merchants can log in to [Waffo Portal](https://dashboard.waffo.com) to view contracted payment methods (Home → Service → Pay-in).

## Advanced Configuration

### Custom HTTP Transport

```go
import (
    "github.com/waffo-com/waffo-go/net"
)

type CustomTransport struct {
    // Your HTTP client
}

func (t *CustomTransport) Send(req *net.HttpRequest) (*net.HttpResponse, error) {
    // Custom implementation
}

cfg, _ := config.NewConfigBuilder().
    APIKey("your-api-key").
    // ... other options
    HttpTransport(&CustomTransport{}).
    Build()
```

### TLS Security Configuration

The SDK enforces TLS 1.2+ by default. No additional configuration is required.

```go
// TLS 1.2+ is enforced automatically in DefaultHttpTransport
transport := net.NewDefaultHttpTransport(timeout)
```

## Handling New API Fields (ExtraParams)

When Waffo API adds new fields that are not yet defined in the SDK, you can use the ExtraParams feature to access these fields without waiting for an SDK update.

### Reading Unknown Fields from Responses

```go
// In Go, use the ExtraParams field (map[string]interface{}) on response data structs
// Note: ExtraParams is populated for any JSON fields not defined in the struct

// Get extra field from response
resp, err := client.Order().Inquiry(ctx, &order.InquiryOrderParams{
    PaymentRequestID: "REQ001",
}, nil)
if resp.IsSuccess() {
    data := resp.GetData()
    // Access extra fields via ExtraParams map
    if newField, ok := data.ExtraParams["newField"]; ok {
        fmt.Printf("New field: %v\n", newField)
    }
}
```

### Sending Extra Fields in Requests

```go
// In Go, use the ExtraParams field (types.ExtraParams = map[string]interface{})
resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    PaymentRequestID: "REQ001",
    MerchantOrderID:  "ORDER001",
    // ... other required fields
    ExtraParams: types.ExtraParams{
        "newField": "value",                      // Extra field
        "nested":   map[string]interface{}{"key": 123}, // Nested object
    },
}, nil)
```

### Important Notes

> **Upgrade SDK Promptly**
>
> ExtraParams is designed as a **temporary solution** for accessing new API fields before SDK updates.
>
> **Best Practices:**
> 1. Check SDK release notes regularly for new field support
> 2. Once SDK officially supports the field, migrate from `getExtraParam("field")` to the official getter (e.g., `getField()`)
> 3. The SDK logs a warning when you use `getExtraParam()` on officially supported fields
>
> **Why migrate?**
> - Official getters provide type safety
> - Better IDE auto-completion and documentation
> - Reduced risk of typos in field names

## Error Handling

### Error Types

| Error Type | Error Code | Description |
|------------|------------|-------------|
| `WaffoError` | `S0002` | Configuration error |
| `WaffoError` | `S0003` | Signature error |
| `WaffoError` | `S0004` | Invalid private key |
| `WaffoError` | `S0005` | Invalid public key |
| `WaffoError` | `S0006` | JSON serialization error |
| `WaffoError` | `S0007` | JSON deserialization error |
| `WaffoUnknownStatusError` | `S0001` | Network error, unknown status |
| `WaffoUnknownStatusError` | `E0001` | API returned error |

### Error Handling Example

```go
import (
    "github.com/waffo-com/waffo-go/errors"
)

resp, err := client.Order().Create(ctx, params, nil)
if err != nil {
    if waffoErr, ok := err.(*errors.WaffoError); ok {
        fmt.Printf("Waffo Error [%s]: %s\n", waffoErr.ErrorCode, waffoErr.Message)
    }
    if unknownErr, ok := err.(*errors.WaffoUnknownStatusError); ok {
        // Idempotent retry may be needed
        fmt.Printf("Unknown Status [%s]: %s\n", unknownErr.ErrorCode, unknownErr.Message)
    }
    return
}

if !resp.IsSuccess() {
    fmt.Printf("API Error: %s - %s\n", resp.ResultCode, resp.ResultMsg)
}
```

## Testing

### Unit Tests

Run unit tests with:

```bash
cd packages/waffo-go
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

### E2E Tests

E2E tests verify the SDK's integration with the real Waffo API using browser automation.

#### Prerequisites

1. Install Playwright browsers:
```bash
go run github.com/playwright-community/playwright-go/cmd/playwright install
```

2. Configure test credentials:
```bash
# Copy the example config
cp test/e2e/application-test.yml.example test/e2e/application-test.yml

# Edit with your sandbox credentials
vim test/e2e/application-test.yml
```

#### Running E2E Tests

```bash
# Run E2E tests (headless mode)
go test -tags=e2e ./test/e2e/... -v

# Run with visible browser (for debugging)
E2E_HEADLESS=false go test -tags=e2e ./test/e2e/... -v

# Run with screenshots
E2E_SCREENSHOT=true go test -tags=e2e ./test/e2e/... -v

# Run specific test
go test -tags=e2e ./test/e2e/... -run TestPaymentFlow -v
```

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E_HEADLESS` | `true` | Run browser in headless mode |
| `E2E_SCREENSHOT` | `false` | Save screenshots during tests |
| `E2E_SCREENSHOT_PATH` | `test-screenshots` | Screenshot save directory |
| `E2E_TIMEOUT` | `30000` | Default timeout in milliseconds |
| `E2E_BROWSER` | `firefox` | Browser type (firefox/chromium/webkit) |
| `WAFFO_E2E_CONFIG` | - | Path to config file |

#### Test Card

Use the following test card in sandbox environment:

| Field | Value |
|-------|-------|
| Card Number | 4111 1111 1111 1111 |
| Expiry | 09/28 |
| CVV | 123 |
| Holder | Tom Clause |

## Support

- 📧 Email: sdk-support@waffo.com
- 📚 Documentation: [https://docs.waffo.com](https://docs.waffo.com)
- 🐛 Issues: [GitHub Issues](https://github.com/waffo-com/waffo-sdk/issues)

## License

[MIT License](LICENSE)

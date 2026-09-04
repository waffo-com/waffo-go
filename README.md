# Waffo Go SDK

<!-- Synced with waffo-sdk/README.md @ commit 53f0d42 -->

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
    "fmt"
    "log"
    "time"

    "github.com/waffo-com/waffo-go/types/order"
)

ctx := context.Background()

// Generate idempotency key (max 32 chars)
paymentRequestID := fmt.Sprintf("%032x", time.Now().UnixNano())
// Important: Persist this ID to database for retry and query

resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    PaymentRequestID: paymentRequestID,
    MerchantOrderID:  fmt.Sprintf("ORDER_%d", time.Now().UnixMilli()),
    OrderCurrency:    "HKD",
    OrderAmount:      "100.00",
    OrderDescription: "Test Product",
    NotifyURL:        "https://your-site.com/webhook",
    UserInfo: &order.UserInfo{
        UserID:       "user_123",
        UserEmail:    "user@example.com",
        UserTerminal: "WEB",
    },
    PaymentInfo: &order.PaymentInfo{
        ProductName: "ONE_TIME_PAYMENT",
    },
    GoodsInfo: &order.GoodsInfo{
        GoodsURL: "https://your-site.com/product/001",
    },
}, nil)

if err != nil {
    log.Fatal(err)
}

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Redirect URL: %s\n", data.FetchRedirectURL())
    fmt.Printf("Acquiring Order ID: %s\n", data.AcquiringOrderID)
} else {
    fmt.Printf("Error: %s - %s\n", resp.GetCode(), resp.GetMessage())
}
```

### 3. Handle Webhook Notifications

```go
import (
    "github.com/waffo-com/waffo-go/core"
)

handler := client.Webhook().
    OnPayment(func(n *core.PaymentNotification) {
        if n.Result != nil {
            fmt.Printf("Payment %s: %s\n", n.Result.AcquiringOrderID, n.Result.OrderStatus)
        }
    }).
    OnRefund(func(n *core.RefundNotification) {
        if n.Result != nil {
            fmt.Printf("Refund %s: %s\n", n.Result.RefundRequestID, n.Result.RefundStatus)
        }
    })

// In your HTTP handler
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-SIGNATURE")

    result := handler.HandleWebhook(string(body), signature)

    if result.Success {
        responseBody, responseSig := handler.BuildSuccessResponse()
        w.Header().Set("X-SIGNATURE", responseSig)
        w.Write([]byte(responseBody))
    } else {
        responseBody, responseSig := handler.BuildFailedResponse(result.Error)
        w.Header().Set("X-SIGNATURE", responseSig)
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
| `ConnectTimeout` | `int64` | ❌ | `10000` | Connection timeout in milliseconds |
| `ReadTimeout` | `int64` | ❌ | `30000` | Read timeout in milliseconds |

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
        signature := c.GetHeader("X-SIGNATURE")

        result := handler.HandleWebhook(string(body), signature)

        if result.Success {
            responseBody, responseSig := handler.BuildSuccessResponse()
            c.Header("X-SIGNATURE", responseSig)
            c.String(http.StatusOK, responseBody)
        } else {
            responseBody, responseSig := handler.BuildFailedResponse(result.Error)
            c.Header("X-SIGNATURE", responseSig)
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
        signature := c.Request().Header.Get("X-SIGNATURE")

        result := handler.HandleWebhook(string(body), signature)

        if result.Success {
            responseBody, responseSig := handler.BuildSuccessResponse()
            c.Response().Header().Set("X-SIGNATURE", responseSig)
            return c.String(http.StatusOK, responseBody)
        }
        responseBody, responseSig := handler.BuildFailedResponse(result.Error)
        c.Response().Header().Set("X-SIGNATURE", responseSig)
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
        signature := c.Get("X-SIGNATURE")

        result := handler.HandleWebhook(string(body), signature)

        if result.Success {
            responseBody, responseSig := handler.BuildSuccessResponse()
            c.Set("X-SIGNATURE", responseSig)
            return c.SendString(responseBody)
        }
        responseBody, responseSig := handler.BuildFailedResponse(result.Error)
        c.Set("X-SIGNATURE", responseSig)
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
    OrderDescription: "Premium Plan",
    NotifyURL:        "https://example.com/webhook",
    UserInfo: &order.UserInfo{
        UserID: "USER-123",
    },
    PaymentInfo: &order.PaymentInfo{
        ProductName: "ONE_TIME_PAYMENT",
    },
}, nil)
```

#### Query Order

```go
resp, err := client.Order().Inquiry(ctx, &order.InquiryOrderParams{
    PaymentRequestID: "REQ-123",
}, nil)

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Order Status: %s\n", data.OrderStatus)
}
```

#### Cancel Order

```go
resp, err := client.Order().Cancel(ctx, &order.CancelOrderParams{
    PaymentRequestID: "REQ-123",
}, nil)
```

### Subscription Management

#### Create Subscription

```go
resp, err := client.Subscription().Create(ctx, &subscription.CreateSubscriptionParams{
    SubscriptionRequest:    "SUB-REQ-123",
    MerchantSubscriptionID: "MERCHANT-SUB-123",
    Amount:                 "9.99",
    Currency:               "USD",
    ProductInfo: &subscription.ProductInfo{
        Description:    "Premium Plan",
        PeriodType:     "MONTHLY",
        PeriodInterval: "1",
    },
    UserInfo: &subscription.SubscriptionUserInfo{
        UserID: "USER-123",
    },
    PaymentInfo: &subscription.SubscriptionPaymentInfo{
        ProductName: "SUBSCRIPTION",
    },
    NotifyURL: "https://example.com/webhook",
}, nil)
```

#### Query Subscription

```go
resp, err := client.Subscription().Inquiry(ctx, &subscription.InquirySubscriptionParams{
    SubscriptionRequest: "SUB-REQ-123",
}, nil)
```

#### Cancel Subscription

```go
resp, err := client.Subscription().Cancel(ctx, &subscription.CancelSubscriptionParams{
    SubscriptionID: "SUB-123",
}, nil)
```

<!-- sdk-sync:start:Subscription%20Change%20(Upgrade%2FDowngrade) -->
### Subscription Change (Upgrade/Downgrade)

Change an existing subscription to a new plan (upgrade or downgrade).

#### Change Subscription

```go
// New subscription request ID for the change
subscriptionRequest := fmt.Sprintf("%032x", time.Now().UnixNano())
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
            Description:    "Annual Premium Subscription (12 months)",
            PeriodType:     "MONTHLY",
            PeriodInterval: "12",
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
<!-- sdk-sync:end:Subscription%20Change%20(Upgrade%2FDowngrade) -->

### Refund Query

```go
resp, err := client.Refund().Inquiry(ctx, &refund.InquiryRefundParams{
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

<!-- sdk-sync:start:Query%20Available%20Payment%20Methods -->
#### Query Available Payment Methods

The `payMethodDetails` response includes the payment method category, supported currency, order amount limits, and refund policy. `currentStatus` can be absent when real-time availability is temporarily unavailable.

```go
import "github.com/waffo-com/waffo-go/types/merchant"

// merchantId is auto-injected from config if MerchantID is set
resp, err := client.PayMethodConfig().Inquiry(ctx, &merchant.InquiryPayMethodConfigParams{}, nil)
if err != nil {
    log.Fatal(err)
}
if resp.IsSuccess() {
    data := resp.GetData()
    for _, detail := range data.PayMethodDetails {
        refundAllowed := "not provided"
        if detail.RefundAllowed != nil {
            refundAllowed = fmt.Sprint(*detail.RefundAllowed)
        }
        fmt.Printf("Payment Method: %s, Type: %s, Country: %s, Currency: %s, Min: %s, Max: %s, Refund policy: %s, Current status: %s\n",
            detail.PayMethodName, detail.PayMethodType, detail.Country, detail.Currency,
            detail.OrderAmountMin, detail.OrderAmountMax, refundAllowed, detail.CurrentStatus)
    }
} else {
    log.Printf("Pay method config inquiry failed: %s - %s", resp.GetCode(), resp.GetMessage())
}
```
<!-- sdk-sync:end:Query%20Available%20Payment%20Methods -->

### Wallet Address Inquiry (x402)

Inquire the on-chain stablecoin receiving address for an x402 payment. The response returns the target network, the on-chain token contract, and the Waffo platform wallet address to send funds to. Phase 1 supports `USDC`.

```go
import "github.com/waffo-com/waffo-go/types/wallet"

// PaymentRequestID: unique idempotency key; OrderCurrency: Phase 1 supports USDC
// MerchantInfo is auto-injected from config when MerchantID is set
resp, err := client.Wallet().Inquiry(ctx, &wallet.WalletInquiryParams{
    PaymentRequestID: "unique-request-id",
    OrderCurrency:    "USDC",
}, nil)
if err != nil {
    log.Fatal(err)
}
if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Network: %s\n", data.Network) // CAIP-2 network id, e.g. eip155:8453 (Base mainnet)
    fmt.Printf("Asset: %s\n", data.Asset)      // on-chain token contract address
    fmt.Printf("Pay to: %s\n", data.PayTo)     // Waffo platform receiving wallet address
} else {
    fmt.Printf("Wallet inquiry failed: %s %s\n", resp.GetCode(), resp.GetMessage())
}
```

#### Wallet Inquiry Response Fields

| Field | Description |
|-------|-------------|
| `network` | CAIP-2 network id (Phase 1: `eip155:8453`, Base mainnet) |
| `asset` | On-chain token contract address for the network + currency |
| `payTo` | Waffo platform receiving wallet address (send funds here) |

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

<!-- sdk-sync:start:Webhook%20Notification%20Payload%20Examples -->
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

For x402 stablecoin payments, `PAYMENT_NOTIFICATION.result` also includes `x402Info` with `x402Request`, `paymentSignatureHeader`, and `x402Response`.

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
<!-- sdk-sync:end:Webhook%20Notification%20Payload%20Examples -->

<!-- sdk-sync:start:Payment%20Method%20Types -->
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

Specify a type and let the user choose on the checkout page:

```json
{"paymentInfo": {"payMethodType": "CREDITCARD"}}
```

Specify an exact payment method:

```json
{"paymentInfo": {"payMethodType": "CREDITCARD", "payMethodName": "CC_VISA"}}
```

Combine multiple types:

```json
{"paymentInfo": {"payMethodType": "CREDITCARD,DEBITCARD"}}
```

Select an e-wallet channel:

```json
{"paymentInfo": {"payMethodType": "EWALLET", "payMethodName": "GCASH"}}
```

> **Note**: For available `ProductName`, `PayMethodType`, `PayMethodName` values, merchants can log in to [Waffo Portal](https://dashboard.waffo.com) to view contracted payment methods (Home → Service → Pay-in).
<!-- sdk-sync:end:Payment%20Method%20Types -->

## Advanced Configuration

### Custom HTTP Transport

```go
import (
    "context"

    "github.com/waffo-com/waffo-go/net"
)

type CustomTransport struct {
    // Your HTTP client
}

func (t *CustomTransport) Send(ctx context.Context, req *net.HttpRequest) (*net.HttpResponse, error) {
    // Custom implementation
    return nil, nil
}

cfg, _ := config.NewConfigBuilder().
    APIKey("your-api-key").
    // ... other options
    CustomTransport(&CustomTransport{}).
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

<!-- sdk-sync:start:Error%20Code%20Classification -->
### Error Code Classification

Error codes are classified by first letter:

| Prefix | Category | Description |
|--------|----------|-------------|
| **S** | SDK Internal Error | SDK client internal error such as network timeout, signing failure, etc. |
| **A** | Merchant Related | Parameter, signature, permission, contract issues on merchant side |
| **B** | User Related | User status, balance, authorization issues |
| **C** | System Related | Waffo system or payment channel issues |
| **D** | Risk Related | Risk control rejection |
| **E** | Unknown Status | Server returned unknown status |
<!-- sdk-sync:end:Error%20Code%20Classification -->

<!-- sdk-sync:start:Complete%20Error%20Code%20Table -->
### Complete Error Code Table

#### SDK Internal Errors (Sxxxx)

| Code | Description | Exception Type | Handling Suggestion |
|------|-------------|----------------|---------------------|
| `S0001` | Network Error | `WaffoUnknownStatusError` | **Status unknown**, need to query order to confirm |
| `S0002` | Invalid Public Key | `WaffoError` | Check if public key is valid Base64 encoded X509 format |
| `S0003` | RSA Signing Failed | `WaffoError` | Check if private key format is correct |
| `S0004` | Response Signature Verification Failed | `ApiResponse.error()` | Check Waffo public key config, contact Waffo |
| `S0005` | Request Serialization Failed | `ApiResponse.error()` | Check request parameter format |
| `S0006` | SDK Unknown Error | `ApiResponse.error()` | Check logs, contact technical support |
| `S0007` | Invalid Private Key | `WaffoError` | Check if private key is valid Base64 encoded PKCS8 format |

> **Important**: `S0001` and `E0001` (returned by server) indicate **unknown status**. Do not close order directly! Should call query API or wait for webhook to confirm actual status.

#### Merchant Related Errors (Axxxxx)

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `0` | Success | 200 |
| `A0001` | Invalid API Key | 401 |
| `A0002` | Invalid Signature | 401 |
| `A0003` | Parameter Validation Failed | 400 |
| `A0004` | Insufficient Permission | 401 |
| `A0005` | Merchant Limit Exceeded | 400 |
| `A0006` | Merchant Status Abnormal | 400 |
| `A0007` | Unsupported Transaction Currency | 400 |
| `A0008` | Transaction Amount Exceeded | 400 |
| `A0009` | Order Not Found | 400 |
| `A0010` | Merchant Contract Does Not Allow This Operation | 400 |
| `A0011` | Idempotent Parameter Mismatch | 400 |
| `A0012` | Merchant Account Insufficient Balance | 400 |
| `A0013` | Order Already Paid, Cannot Cancel | 400 |
| `A0014` | Refund Rules Do Not Allow Refund | 400 |
| `A0015` | Payment Channel Does Not Support Cancel | 400 |
| `A0016` | Payment Channel Rejected Cancel | 400 |
| `A0017` | Payment Channel Does Not Support Refund | 400 |
| `A0018` | Payment Method Does Not Match Merchant Contract | 400 |
| `A0019` | Cannot Refund Due to Chargeback Dispute | 400 |
| `A0020` | Payment Amount Exceeds Single Transaction Limit | 400 |
| `A0021` | Cumulative Payment Amount Exceeds Daily Limit | 400 |
| `A0022` | Multiple Products Exist, Need to Specify Product Name | 400 |
| `A0023` | Token Expired, Cannot Create Order | 400 |
| `A0024` | Exchange Rate Expired, Cannot Process Order | 400 |
| `A0026` | Unsupported Checkout Language | 400 |
| `A0027` | Refund Count Reached Limit (50 times) | 400 |
| `A0029` | Invalid Card Data Provided by Merchant | 400 |
| `A0030` | Card BIN Not Found | 400 |
| `A0031` | Unsupported Card Scheme or Card Type | 400 |
| `A0032` | Invalid Payment Token Data | 400 |
| `A0033` | Multiple Payment Methods with Same Name, Need to Specify Country | 400 |
| `A0034` | Order Expiry Time Provided by Merchant Has Passed | 400 |
| `A0035` | Current Order Does Not Support Capture Operation | 400 |
| `A0036` | Current Order Status Does Not Allow Capture Operation | 400 |
| `A0037` | User Payment Token Invalid or Expired | 400 |
| `A0038` | MIT Transaction Requires Verified User Payment Token | 400 |
| `A0039` | Order Already Refunded by Chargeback Prevention Service | 400 |
| `A0040` | Order Cannot Be Created Concurrently | 400 |
| `A0045` | MIT Transaction Cannot Process, tokenId Status Unverified | 400 |

#### User Related Errors (Bxxxxx)

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `B0001` | User Status Abnormal | 400 |
| `B0002` | User Limit Exceeded | 400 |
| `B0003` | User Insufficient Balance | 400 |
| `B0004` | User Did Not Pay Within Timeout | 400 |
| `B0005` | User Authorization Failed | 400 |
| `B0006` | Invalid Phone Number | 400 |
| `B0007` | Invalid Email Format | 400 |

#### System Related Errors (Cxxxxx)

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `C0001` | System Error | 500 |
| `C0002` | Merchant Contract Invalid | 500 |
| `C0003` | Order Status Invalid, Cannot Continue Processing | 500 |
| `C0004` | Order Information Mismatch | 500 |
| `C0005` | Payment Channel Rejected | 503 |
| `C0006` | Payment Channel Error | 503 |
| `C0007` | Payment Channel Under Maintenance | 503 |

#### Risk Related Errors (Dxxxxx)

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `D0001` | Risk Control Rejected | 406 |

#### Unknown Status Errors (Exxxxx)

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `E0001` | Unknown Status (Need to query or wait for callback) | 500 |

> **Note**: When receiving `E0001` error code, it indicates transaction status is unknown. **Do not close order directly**, should call query API to confirm actual status, or wait for webhook callback notification.
<!-- sdk-sync:end:Complete%20Error%20Code%20Table -->

<!-- sdk-sync:start:Environment%20URLs -->
### Environment URLs

| Environment | Base URL | Description |
|-------------|----------|-------------|
| `SANDBOX` | `https://api-sandbox.waffo.com` | Test environment |
| `PRODUCTION` | `https://api.waffo.com` | Production environment |
<!-- sdk-sync:end:Environment%20URLs -->

<!-- sdk-sync:start:Timeout%20Configuration%20Recommendations -->
### Timeout Configuration Recommendations

| Operation Type | Connect Timeout | Read Timeout | Notes |
|----------------|-----------------|--------------|-------|
| Create Order | 5s | 30s | Recommended |
| Create Subscription | 5s | 30s | Recommended |
| Refund Operation | 5s | 30s | Recommended |
| Query Operations | 5s | 15s | Can be shorter |
<!-- sdk-sync:end:Timeout%20Configuration%20Recommendations -->

<!-- sdk-sync:start:Connection%20Pool%20Recommendations -->
### Connection Pool Recommendations

| Scenario | Max Connections | Max Per Route | Notes |
|----------|-----------------|---------------|-------|
| Low Traffic (< 10 QPS) | 20 | 10 | Default config sufficient |
| Medium Traffic (10-100 QPS) | 50 | 20 | Consider using OkHttp |
| High Traffic (> 100 QPS) | 100-200 | 50 | Consider Apache HttpClient |
<!-- sdk-sync:end:Connection%20Pool%20Recommendations -->

<!-- sdk-sync:start:Update%20Subscription -->
#### Update Subscription

Update the billing amounts of an active subscription without re-authorization. Provide `subscriptionRequest` or `subscriptionId`. To charge a seat add-on immediately, include `topupInfo`; the updated billing amounts take effect from the next billing period.

```go
resp, err := client.Subscription().Update(ctx, &subscription.UpdateSubscriptionParams{
    SubscriptionID: "subscription_id",
    Amount:         "129.00", // New per-period amount from the next billing period
    ProductInfo: &subscription.UpdateProductInfo{
        TrialPeriodAmount: "0",
        ScheduledAmounts: []subscription.ScheduledAmount{
            {Period: "2", Amount: "99.00"}, // Per-period override
            {Period: "3", Amount: "129.00"},
        },
    },
    TopupInfo: &subscription.SubscriptionUpdateTopupInfo{
        TopupRequest:   "mimo-topup-001", // Idempotency key for this top-up
        TopupAmount:    "30.00",           // Amount charged immediately
        Description:    "Seat add-on 3 seats",
        OrderExpiredAt: "2030-01-01T00:00:00Z",
    },
}, nil)
if err != nil {
    log.Fatal(err)
}
if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("Previous Amount: %s\n", data.PreviousAmount)
    fmt.Printf("New Amount: %s\n", data.NewAmount)
    fmt.Printf("Next Effective Period: %s\n", data.NextEffectivePeriod)
    fmt.Printf("Top-up Order: %s\n", data.TopupInfo.AcquiringOrderID)
    fmt.Printf("Top-up Status: %s\n", data.TopupInfo.TopupStatus)
}
```
<!-- sdk-sync:end:Update%20Subscription -->

## Support

- 📧 Email: sdk-support@waffo.com
- 📚 Documentation: [https://docs.waffo.com](https://docs.waffo.com)
- 🐛 Issues: [GitHub Issues](https://github.com/waffo-com/waffo-sdk/issues)

## License

[MIT License](LICENSE)

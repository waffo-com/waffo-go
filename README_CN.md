# Waffo Go SDK

<!-- Synced with waffo-sdk/README_CN.md @ commit ee83ead -->

<!-- Synced with waffo-sdk/README_CN.md -->

[English](README.md) | **中文**

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Waffo 支付平台](https://www.waffo.com) 官方 Go SDK，为 AI 产品、SaaS 服务等提供一站式全球支付解决方案。

## 简介

### 核心功能

- **全球支付**：支持信用卡、借记卡、电子钱包、虚拟账户等多种支付方式，覆盖全球主流支付渠道
- **订阅管理**：完整的订阅生命周期管理，支持试用期、周期性扣款、订阅升降级
- **退款处理**：灵活的全额/部分退款能力，支持退款状态追踪
- **Webhook 通知**：支付结果、退款、订阅状态变更等实时推送通知
- **安全可靠**：PCI DSS 认证，RSA 签名验证，强制 TLS 1.2+ 加密传输

### 适用场景

| 场景 | 描述 |
|------|------|
| **AI 产品** | ChatGPT 类应用、AI 写作工具、AI 绘画，按量计费或订阅制 |
| **SaaS 服务** | 企业软件订阅、在线协作工具、云服务，周期性支付 |
| **内容平台** | 会员订阅、付费内容、打赏场景 |

## 目录

- [环境要求](#环境要求)
- [安装](#安装)
- [快速开始](#快速开始)
- [配置](#配置)
  - [框架集成](#框架集成)
- [API 使用](#api-使用)
  - [订单管理](#订单管理)
  - [订阅管理](#订阅管理)
  - [退款查询](#退款查询)
  - [商户配置](#商户配置)
- [高级配置](#高级配置)
  - [自定义 HTTP 传输层](#自定义-http-传输层)
  - [TLS 安全配置](#tls-安全配置)
- [Webhook 处理](#webhook-处理)
- [支付方式类型](#支付方式类型)
- [处理新增 API 字段 (ExtraParams)](#处理新增-api-字段-extraparams)
- [错误处理](#错误处理)
- [支持](#支持)
- [许可证](#许可证)

## 环境要求

- Go 1.20+
- 无外部依赖（仅使用标准库）

### 版本兼容性

| Go 版本 | 支持状态 |
|---------|----------|
| 1.22.x | ✅ 完全支持 |
| 1.21.x | ✅ 完全支持（推荐） |
| 1.20.x | ✅ 完全支持 |
| < 1.20 | ❌ 不支持 |

## 安装

```bash
go get github.com/waffo-com/waffo-sdk/packages/waffo-go
```

## 快速开始

### 1. 初始化 SDK

```go
package main

import (
    "github.com/waffo-com/waffo-sdk/packages/waffo-go"
    "github.com/waffo-com/waffo-sdk/packages/waffo-go/config"
)

func main() {
    cfg, err := config.NewConfigBuilder().
        APIKey("your-api-key").
        PrivateKey("your-base64-encoded-private-key").
        WaffoPublicKey("waffo-public-key").           // 从 Waffo 控制台获取
        MerchantID("your-merchant-id").               // 自动注入到请求中
        Environment(config.Sandbox).                   // Sandbox 或 Production
        Build()
    if err != nil {
        log.Fatal(err)
    }

    client := waffo.New(cfg)
}
```

### 2. 创建支付订单

```go
import (
    "context"
    "github.com/google/uuid"
    "github.com/waffo-com/waffo-sdk/packages/waffo-go/types/order"
)

ctx := context.Background()

// 生成幂等键（最大32字符）
paymentRequestID := uuid.New().String()[:32]

resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    MerchantOrderID:  "ORDER-" + uuid.New().String()[:8],
    PaymentRequestID: paymentRequestID,
    OrderAmount:      "99.99",
    OrderCurrency:    "USD",
    Subject:          "高级订阅",
    PaymentMethod:    "CARD",
    // ... 其他参数
}, nil)

if err != nil {
    log.Fatal(err)
}

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("支付链接: %s\n", data.PaymentURL)
} else {
    fmt.Printf("错误: %s - %s\n", resp.ResultCode, resp.ResultMsg)
}
```

### 3. 处理 Webhook 通知

```go
import (
    "github.com/waffo-com/waffo-sdk/packages/waffo-go/core"
)

handler := client.Webhook().
    OnPayment(func(n *core.PaymentNotification) {
        fmt.Printf("支付 %s: %s\n", n.PaymentRequestID, n.OrderStatus)
    }).
    OnRefund(func(n *core.RefundNotification) {
        fmt.Printf("退款 %s: %s\n", n.RefundRequestID, n.RefundStatus)
    })

// 在 HTTP 处理器中
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

## 配置

### 配置选项

| 选项 | 类型 | 必填 | 默认值 | 描述 |
|------|------|------|--------|------|
| `APIKey` | `string` | ✅ | - | 从 Waffo 控制台获取的 API Key |
| `PrivateKey` | `string` | ✅ | - | Base64 编码的 PKCS#8 私钥 |
| `WaffoPublicKey` | `string` | ✅ | - | Base64 编码的 X.509 公钥（Waffo 提供） |
| `MerchantID` | `string` | ❌ | - | 商户 ID，用于自动注入 |
| `Environment` | `Environment` | ❌ | `Sandbox` | `Sandbox` 或 `Production` |
| `Timeout` | `time.Duration` | ❌ | `30s` | 请求超时时间 |

### 环境变量

```go
// 从环境变量初始化
client, err := waffo.FromEnv()
```

| 变量 | 描述 |
|------|------|
| `WAFFO_API_KEY` | API Key（必填） |
| `WAFFO_PRIVATE_KEY` | Base64 编码的私钥（必填） |
| `WAFFO_PUBLIC_KEY` | Base64 编码的 Waffo 公钥（必填） |
| `WAFFO_ENVIRONMENT` | `SANDBOX` 或 `PRODUCTION`（默认：SANDBOX） |
| `WAFFO_MERCHANT_ID` | 商户 ID（可选） |

### 框架集成

#### Gin

```go
import (
    "github.com/gin-gonic/gin"
)

func setupRoutes(r *gin.Engine, waffoClient *waffo.Waffo) {
    handler := waffoClient.Webhook().
        OnPayment(func(n *core.PaymentNotification) {
            // 处理支付通知
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
            // 处理支付通知
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
            // 处理支付通知
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

## API 使用

### 订单管理

#### 创建订单

```go
resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    MerchantOrderID:  "ORDER-123",
    PaymentRequestID: "REQ-123",
    OrderAmount:      "99.99",
    OrderCurrency:    "USD",
    Subject:          "高级套餐",
    PaymentMethod:    "CARD",
    ReturnURL:        "https://example.com/return",
    NotifyURL:        "https://example.com/webhook",
}, nil)
```

#### 查询订单

```go
resp, err := client.Order().Query(ctx, &order.QueryOrderParams{
    PaymentRequestID: "REQ-123",
}, nil)

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("订单状态: %s\n", data.OrderStatus)
}
```

#### 关闭订单

```go
resp, err := client.Order().Close(ctx, &order.CloseOrderParams{
    PaymentRequestID: "REQ-123",
}, nil)
```

### 订阅管理

#### 创建订阅

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

#### 查询订阅

```go
resp, err := client.Subscription().Query(ctx, &subscription.QuerySubscriptionParams{
    SubscriptionRequest: "SUB-REQ-123",
}, nil)
```

#### 取消订阅

```go
resp, err := client.Subscription().Cancel(ctx, &subscription.CancelSubscriptionParams{
    SubscriptionRequest: "SUB-REQ-123",
}, nil)
```

#### 订阅变更示例

变更现有订阅到新的计划（升级或降级）。

<!-- tabs:start -->
#### **Node.js**

```typescript
import { randomUUID } from 'crypto';
import { WaffoUnknownStatusError } from '@waffo/waffo-node';

// 新订阅请求 ID
const subscriptionRequest = randomUUID().replace(/-/g, '');
const originSubscriptionRequest = 'original-subscription-request-id';

try {
  const response = await waffo.subscription().change({
    subscriptionRequest,
    originSubscriptionRequest,
    remainingAmount: '50.00',  // 原订阅剩余价值
    currency: 'HKD',
    requestedAt: new Date().toISOString(),
    notifyUrl: 'https://your-site.com/webhook/subscription',
    productInfoList: [
      {
        description: '年度高级订阅',
        periodType: 'YEAR',
        periodInterval: '1',
        amount: '999.00',
      },
    ],
    userInfo: {
      userId: 'user_123',
      userEmail: 'user@example.com',
    },
    goodsInfo: {
      goodsId: 'GOODS_PREMIUM',
      goodsName: '高级计划',
    },
    paymentInfo: {
      productName: 'SUBSCRIPTION',
    },
  });

  if (response.isSuccess()) {
    const data = response.getData();
    console.log('变更状态:', data.subscriptionChangeStatus);
    console.log('新订阅 ID:', data.subscriptionId);

    // 处理不同状态
    if (data.subscriptionChangeStatus === 'AUTHORIZATION_REQUIRED') {
      // 用户需要授权
      const action = JSON.parse(data.subscriptionAction);
      console.log('重定向用户到:', action.webUrl);
    } else if (data.subscriptionChangeStatus === 'SUCCESS') {
      console.log('订阅升级成功');
    }
  }
} catch (error) {
  if (error instanceof WaffoUnknownStatusError) {
    // 状态未知 - 不要假设失败！用户可能已完成支付
    console.error('状态未知，需要查询确认:', error.message);

    // 正确处理：调用查询 API 确认实际状态
    const inquiryResponse = await waffo.subscription().changeInquiry({
      subscriptionRequest,
      originSubscriptionRequest,
    });
    // 或等待 Webhook 回调通知
  } else {
    throw error;
  }
}
```

#### **Java**

```java
import com.waffo.types.subscription.*;
import com.waffo.types.iso.CurrencyCode;
import java.util.Arrays;
import java.util.UUID;

// 新订阅请求 ID
String subscriptionRequest = UUID.randomUUID().toString().replace("-", "");

SubscriptionChangeParams params = SubscriptionChangeParams.builder()
    .subscriptionRequest(subscriptionRequest)
    .originSubscriptionRequest("original-subscription-request-id")
    .remainingAmount("50.00")  // 原订阅剩余价值
    .currency(CurrencyCode.HKD)
    .requestedAt(Iso8601InstantSerializer.now())
    .notifyUrl("https://your-site.com/webhook/subscription")
    .productInfoList(Arrays.asList(
        SubscriptionChangeProductInfo.builder()
            .description("年度高级订阅")
            .periodType(PeriodType.YEAR)
            .periodInterval("1")
            .amount("999.00")
            .build()
    ))
    .userInfo(SubscriptionUserInfo.builder()
        .userId("user_123")
        .userEmail("user@example.com")
        .build())
    .goodsInfo(SubscriptionGoodsInfo.builder()
        .goodsId("GOODS_PREMIUM")
        .goodsName("高级计划")
        .build())
    .paymentInfo(SubscriptionPaymentInfo.builder()
        .productName("SUBSCRIPTION")
        .build())
    .build();

try {
    ApiResponse<SubscriptionChangeData> response = waffo.subscription().change(params);

    if (response.isSuccess()) {
        SubscriptionChangeData data = response.getData().get();
        System.out.println("变更状态: " + data.getSubscriptionChangeStatus());
        System.out.println("新订阅 ID: " + data.getSubscriptionId());

        // 处理不同状态
        if (data.isAuthorizationRequired()) {
            String redirectUrl = data.fetchRedirectUrl();
            System.out.println("重定向用户到: " + redirectUrl);
        } else if (data.isSuccess()) {
            System.out.println("订阅升级成功");
        }
    }
} catch (WaffoUnknownStatusException e) {
    // 状态未知 - 不要假设失败！用户可能已完成支付
    System.err.println("状态未知，需要查询确认: " + e.getMessage());

    // 正确处理：调用查询 API 确认实际状态
    ApiResponse<SubscriptionChangeData> inquiryResponse = waffo.subscription().changeInquiry(
        SubscriptionChangeInquiryParams.builder()
            .subscriptionRequest(subscriptionRequest)
            .originSubscriptionRequest(originSubscriptionRequest)
            .build()
    );
    // 或等待 Webhook 回调通知
}
```
<!-- tabs:end -->

#### 查询订阅变更状态

<!-- tabs:start -->
#### **Node.js**

```typescript
const response = await waffo.subscription().changeInquiry({
  subscriptionRequest: 'new-subscription-request-id',
  originSubscriptionRequest: 'original-subscription-request-id',
});

if (response.isSuccess()) {
  const data = response.getData();
  console.log('变更状态:', data.subscriptionChangeStatus);
  console.log('新订阅 ID:', data.subscriptionId);
  console.log('剩余金额:', data.remainingAmount);
}
```

#### **Java**

```java
SubscriptionChangeInquiryParams params = SubscriptionChangeInquiryParams.builder()
    .subscriptionRequest("new-subscription-request-id")
    .originSubscriptionRequest("original-subscription-request-id")
    .build();

ApiResponse<SubscriptionChangeInquiryData> response = waffo.subscription().changeInquiry(params);

if (response.isSuccess()) {
    SubscriptionChangeInquiryData data = response.getData().get();
    System.out.println("变更状态: " + data.getSubscriptionChangeStatus());
    System.out.println("新订阅 ID: " + data.getSubscriptionId());
    System.out.println("剩余金额: " + data.getRemainingAmount());
}
```
<!-- tabs:end -->

#### 订阅变更状态值

| 状态 | 说明 |
|------|------|
| `IN_PROGRESS` | 变更处理中 |
| `AUTHORIZATION_REQUIRED` | 需要用户授权（重定向到 webUrl） |
| `SUCCESS` | 变更成功完成 |
| `CLOSED` | 变更已关闭（超时或失败） |

### 退款查询

```go
resp, err := client.Refund().Query(ctx, &refund.QueryRefundParams{
    RefundRequestID: "REFUND-123",
}, nil)

if resp.IsSuccess() {
    data := resp.GetData()
    fmt.Printf("退款状态: %s\n", data.RefundStatus)
}
```

### 商户配置

```go
resp, err := client.MerchantConfig().Inquiry(ctx, &merchant.InquiryMerchantConfigParams{
    // 参数
}, nil)
```

## 高级配置

### 自定义 HTTP 传输层

```go
import (
    "github.com/waffo-com/waffo-sdk/packages/waffo-go/net"
)

type CustomTransport struct {
    // 你的 HTTP 客户端
}

func (t *CustomTransport) Send(req *net.HttpRequest) (*net.HttpResponse, error) {
    // 自定义实现
}

cfg, _ := config.NewConfigBuilder().
    APIKey("your-api-key").
    // ... 其他选项
    HttpTransport(&CustomTransport{}).
    Build()
```

### TLS 安全配置

SDK 默认强制使用 TLS 1.2+，无需额外配置。

```go
// DefaultHttpTransport 自动强制使用 TLS 1.2+
transport := net.NewDefaultHttpTransport(timeout)
```

## Webhook 处理

### 链式处理器

```go
handler := client.Webhook().
    OnPayment(func(n *core.PaymentNotification) {
        // 处理支付
    }).
    OnRefund(func(n *core.RefundNotification) {
        // 处理退款
    }).
    OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {
        // 处理订阅状态
    }).
    OnSubscriptionPayment(func(n *core.SubscriptionPaymentNotification) {
        // 处理订阅支付
    }).
    OnSubscriptionPeriodChanged(func(n *core.SubscriptionPeriodChangedNotification) {
        // 处理周期变更
    })
```

### 通知类型

| 事件类型 | 处理方法 | 说明 |
|----------|----------|------|
| `PAYMENT_NOTIFICATION` | `onPayment()` | 支付结果通知（每次支付尝试都会触发，包括重试） |
| `REFUND_NOTIFICATION` | `onRefund()` | 退款结果通知 |
| `SUBSCRIPTION_STATUS_NOTIFICATION` | `onSubscriptionStatus()` | 订阅状态变更通知（订阅主记录状态变化时触发） |
| `SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION` | `onSubscriptionPeriodChanged()` | 订阅周期变更通知（每个周期的最终结果） |
| `SUBSCRIPTION_CHANGE_NOTIFICATION` | `onSubscriptionChange()` | 订阅变更（升降级）结果通知 |

### 订阅通知类型说明

| 通知类型 | 触发条件 | 范围 | 包含重试事件 | 典型用途 |
|----------|----------|------|--------------|----------|
| `SUBSCRIPTION_STATUS_NOTIFICATION` | 订阅主记录状态变化 | 订阅级别 | 否 | 跟踪订阅生命周期：首次支付成功激活(ACTIVE)、取消(MERCHANT_CANCELLED, CHANNEL_CANCELLED)、首次支付失败关闭(CLOSE)等 |
| `SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION` | 订阅周期达到最终状态 | 周期级别 | 否（仅最终结果） | 只需要每个周期的最终结果，不需要中间重试事件 |
| `SUBSCRIPTION_CHANGE_NOTIFICATION` | 订阅变更（升降级）完成 | 变更请求级别 | 否（仅最终结果） | 跟踪订阅变更结果：SUCCESS 或 CLOSED |
| `PAYMENT_NOTIFICATION` | 每次支付订单 | 支付订单级别 | 是（包含所有重试） | 需要每次支付尝试的完整详情，包括失败原因、时间戳、重试详情 |

> **选择指南**:
> - 如果只关心订阅激活/取消，使用 `SUBSCRIPTION_STATUS_NOTIFICATION`
> - 如果只关心每个周期的最终续费结果，使用 `SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION`
> - 如果只关心订阅变更（升降级）的最终结果，使用 `SUBSCRIPTION_CHANGE_NOTIFICATION`
> - 如果需要跟踪每次支付尝试（包括重试），使用 `PAYMENT_NOTIFICATION`

> **订阅变更（升降级）Webhook 说明**:
> 当订阅变更处理完成时，会触发以下通知：
> - `SUBSCRIPTION_CHANGE_NOTIFICATION`: 订阅变更完成时触发（SUCCESS 或 CLOSED）
> - `SUBSCRIPTION_STATUS_NOTIFICATION`: 原订阅状态变更为 `MERCHANT_CANCELLED` 时触发
> - `SUBSCRIPTION_STATUS_NOTIFICATION`: 新订阅状态变更为 `ACTIVE` 时触发
> - `PAYMENT_NOTIFICATION`: 如果升级需要补差价付款时触发

### Webhook 通知载荷示例

以下示例展示各通知类型的实际载荷结构。

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

以上两种通知类型共享相同的载荷结构：

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

## 支付方式类型

### payMethodType 参考

| 类型 | 说明 | 示例 payMethodName |
|------|------|-------------------|
| `CREDITCARD` | 信用卡 | CC_VISA, CC_MASTERCARD, CC_AMEX, CC_JCB 等 |
| `DEBITCARD` | 借记卡 | DC_VISA, DC_MASTERCARD, DC_ELO 等 |
| `EWALLET` | 电子钱包 | GCASH, DANA, PROMPTPAY, GRABPAY 等 |
| `VA` | 虚拟账户 | BCA, BNI, BRI, MANDIRI 等 |
| `APPLEPAY` | Apple Pay | APPLEPAY |
| `GOOGLEPAY` | Google Pay | GOOGLEPAY |

> **注意**: 可用的 `ProductName`、`PayMethodType`、`PayMethodName` 值，商户可登录 [Waffo Portal](https://dashboard.waffo.com) 查看已签约的支付方式（首页 → 服务 → 收单）。

## 处理新增 API 字段 (ExtraParams)

当 Waffo API 新增字段但 SDK 尚未更新时，可以使用 ExtraParams 功能访问这些字段，无需等待 SDK 更新。

### 从响应中读取未知字段

<!-- tabs:start -->
#### **Node.js**

```typescript
// 从响应中获取额外字段
const response = await waffo.order().inquiry({ paymentRequestId: 'REQ001' });
if (response.isSuccess()) {
  const data = response.getData();

  // 访问 SDK 中尚未定义的字段
  const newField = data.extraParams?.['newField'];

  // 如果知道类型，可以使用类型断言
  const typedValue = data.extraParams?.['newField'] as string;
}

// 从 Webhook 通知中获取额外字段
webhookHandler.onPaymentNotification((notification) => {
  const result = notification.result;
  const newField = result.extraParams?.['newField'];
});
```

#### **Java**

```java
// 从响应中获取额外字段
ApiResponse<InquiryOrderData> response = waffo.order().inquiry(params);
if (response.isSuccess()) {
    InquiryOrderData data = response.getData().orElse(null);

    // 访问 SDK 中尚未定义的字段
    Object newField = data.getExtraParam("newField");

    // 类型安全的 getter
    String typedValue = data.getExtraParam("newField", String.class);

    // 检查字段是否存在
    if (data.hasExtraParam("newField")) {
        // ...
    }
}

// 从 Webhook 通知中获取额外字段
PaymentNotificationResult result = notification.getResult();
Object newField = result.getExtraParam("newField");
```
<!-- tabs:end -->

### 在请求中发送额外字段

<!-- tabs:start -->
#### **Node.js**

```typescript
// TypeScript 类型定义包含索引签名 [key: string]: unknown
// 可以直接在任何请求中添加额外字段
const response = await waffo.order().create({
  paymentRequestId: 'REQ001',
  merchantOrderId: 'ORDER001',
  // ... 其他必填字段
  newField: 'value',           // 额外字段 - 无类型错误
  nested: { key: 123 }         // 嵌套对象 - 同样支持
});
```

#### **Java**

```java
// 在请求中添加额外字段
CreateOrderParams params = CreateOrderParams.builder()
    .paymentRequestId("REQ001")
    .merchantOrderId("ORDER001")
    // ... 其他必填字段
    .extraParam("newField", "value")           // 单个字段
    .extraParam("nested", Map.of("key", 123))  // 嵌套对象
    .build();
```
<!-- tabs:end -->

### 重要提示

> **请及时升级 SDK**
>
> ExtraParams 是一个**临时解决方案**，用于在 SDK 更新前访问新增的 API 字段。
>
> **最佳实践：**
> 1. 定期查看 SDK 发布说明，了解新增字段支持
> 2. SDK 正式支持该字段后，请从 `getExtraParam("field")` 迁移到官方 getter（如 `getField()`）
> 3. 当您对已支持的字段使用 `getExtraParam()` 时，SDK 会输出警告日志
>
> **为什么要迁移？**
> - 官方 getter 提供类型安全
> - 更好的 IDE 自动补全和文档支持
> - 减少字段名拼写错误的风险

## 错误处理

### 错误类型

| 错误类型 | 错误码 | 描述 |
|----------|--------|------|
| `WaffoError` | `S0002` | 配置错误 |
| `WaffoError` | `S0003` | 签名错误 |
| `WaffoError` | `S0004` | 无效的私钥 |
| `WaffoError` | `S0005` | 无效的公钥 |
| `WaffoError` | `S0006` | JSON 序列化错误 |
| `WaffoError` | `S0007` | JSON 反序列化错误 |
| `WaffoUnknownStatusError` | `S0001` | 网络错误，状态未知 |
| `WaffoUnknownStatusError` | `E0001` | API 返回错误 |

### 错误处理示例

```go
import (
    "github.com/waffo-com/waffo-sdk/packages/waffo-go/errors"
)

resp, err := client.Order().Create(ctx, params, nil)
if err != nil {
    if waffoErr, ok := err.(*errors.WaffoError); ok {
        fmt.Printf("Waffo 错误 [%s]: %s\n", waffoErr.ErrorCode, waffoErr.Message)
    }
    if unknownErr, ok := err.(*errors.WaffoUnknownStatusError); ok {
        // 可能需要幂等重试
        fmt.Printf("未知状态 [%s]: %s\n", unknownErr.ErrorCode, unknownErr.Message)
    }
    return
}

if !resp.IsSuccess() {
    fmt.Printf("API 错误: %s - %s\n", resp.ResultCode, resp.ResultMsg)
}
```

### 错误码分类

错误码按首字母分类：

| 前缀 | 类别 | 说明 |
|------|------|------|
| **S** | SDK 内部错误 | SDK 客户端内部错误，如网络超时、签名失败等 |
| **A** | 商户相关 | 参数、签名、权限、合约问题 |
| **B** | 用户相关 | 用户状态、余额、授权问题 |
| **C** | 系统相关 | Waffo 系统或支付渠道问题 |
| **D** | 风控相关 | 风控拒绝 |
| **E** | 未知状态 | 服务器返回未知状态 |

### 完整错误码表

#### SDK 内部错误 (Sxxxx)

| 错误码 | 说明 | 异常类型 | 处理建议 |
|--------|------|----------|----------|
| `S0001` | 网络错误 | `WaffoUnknownStatusError` | **状态未知**，需查询订单确认 |
| `S0002` | 无效公钥 | `WaffoError` | 检查公钥是否为有效的 Base64 编码 X509 格式 |
| `S0003` | RSA 签名失败 | `WaffoError` | 检查私钥格式是否正确 |
| `S0004` | 响应签名验证失败 | `ApiResponse.error()` | 检查 Waffo 公钥配置，联系 Waffo |
| `S0005` | 请求序列化失败 | `ApiResponse.error()` | 检查请求参数格式 |
| `S0006` | SDK 未知错误 | `ApiResponse.error()` | 检查日志，联系技术支持 |
| `S0007` | 无效私钥 | `WaffoError` | 检查私钥是否为有效的 Base64 编码 PKCS8 格式 |

> **重要**: `S0001` 和 `E0001`（服务器返回）表示**未知状态**。不要直接关闭订单！应调用查询 API 或等待 Webhook 确认实际状态。

#### 商户相关错误 (Axxxxx)

| 错误码 | 说明 | HTTP 状态 |
|--------|------|-----------|
| `0` | 成功 | 200 |
| `A0001` | 无效 API Key | 401 |
| `A0002` | 无效签名 | 401 |
| `A0003` | 参数验证失败 | 400 |
| `A0004` | 权限不足 | 401 |
| `A0005` | 商户限额已超 | 400 |
| `A0006` | 商户状态异常 | 400 |
| `A0007` | 不支持的交易币种 | 400 |
| `A0008` | 交易金额超限 | 400 |
| `A0009` | 订单不存在 | 400 |
| `A0010` | 商户合约不允许此操作 | 400 |
| `A0011` | 幂等参数不匹配 | 400 |
| `A0012` | 商户账户余额不足 | 400 |
| `A0013` | 订单已支付，无法取消 | 400 |
| `A0014` | 退款规则不允许退款 | 400 |
| `A0015` | 支付渠道不支持取消 | 400 |
| `A0016` | 支付渠道拒绝取消 | 400 |
| `A0017` | 支付渠道不支持退款 | 400 |
| `A0018` | 支付方式与商户合约不匹配 | 400 |
| `A0019` | 因拒付争议无法退款 | 400 |
| `A0020` | 支付金额超过单笔限额 | 400 |
| `A0021` | 累计支付金额超过日限额 | 400 |
| `A0022` | 存在多个产品，需指定产品名称 | 400 |
| `A0023` | Token 已过期，无法创建订单 | 400 |
| `A0024` | 汇率已过期，无法处理订单 | 400 |
| `A0026` | 不支持的结账语言 | 400 |
| `A0027` | 退款次数已达上限（50次） | 400 |
| `A0029` | 商户提供的卡数据无效 | 400 |
| `A0030` | 卡 BIN 未找到 | 400 |
| `A0031` | 不支持的卡组织或卡类型 | 400 |
| `A0032` | 无效的支付 Token 数据 | 400 |
| `A0033` | 多个同名支付方式，需指定国家 | 400 |
| `A0034` | 商户提供的订单过期时间已过 | 400 |
| `A0035` | 当前订单不支持捕获操作 | 400 |
| `A0036` | 当前订单状态不允许捕获操作 | 400 |
| `A0037` | 用户支付 Token 无效或已过期 | 400 |
| `A0038` | MIT 交易需要已验证的用户支付 Token | 400 |
| `A0039` | 订单已被拒付预防服务退款 | 400 |
| `A0040` | 订单不能并发创建 | 400 |
| `A0045` | MIT 交易无法处理，tokenId 状态未验证 | 400 |

#### 用户相关错误 (Bxxxxx)

| 错误码 | 说明 | HTTP 状态 |
|--------|------|-----------|
| `B0001` | 用户状态异常 | 400 |
| `B0002` | 用户限额已超 | 400 |
| `B0003` | 用户余额不足 | 400 |
| `B0004` | 用户未在超时时间内支付 | 400 |
| `B0005` | 用户授权失败 | 400 |
| `B0006` | 无效的手机号码 | 400 |
| `B0007` | 无效的邮箱格式 | 400 |

#### 系统相关错误 (Cxxxxx)

| 错误码 | 说明 | HTTP 状态 |
|--------|------|-----------|
| `C0001` | 系统错误 | 500 |
| `C0002` | 商户合约无效 | 500 |
| `C0003` | 订单状态无效，无法继续处理 | 500 |
| `C0004` | 订单信息不匹配 | 500 |
| `C0005` | 支付渠道拒绝 | 503 |
| `C0006` | 支付渠道错误 | 503 |
| `C0007` | 支付渠道维护中 | 503 |

#### 风控相关错误 (Dxxxxx)

| 错误码 | 说明 | HTTP 状态 |
|--------|------|-----------|
| `D0001` | 风控拒绝 | 406 |

#### 未知状态错误 (Exxxxx)

| 错误码 | 说明 | HTTP 状态 |
|--------|------|-----------|
| `E0001` | 未知状态（需查询或等待回调） | 500 |

> **注意**: 收到 `E0001` 错误码时，表示交易状态未知。**不要直接关闭订单**，应调用查询 API 确认实际状态，或等待 Webhook 回调通知。

## 支持

- 📧 邮箱: sdk-support@waffo.com
- 📚 文档: [https://docs.waffo.com](https://docs.waffo.com)
- 🐛 问题反馈: [GitHub Issues](https://github.com/waffo-com/waffo-sdk/issues)

## 许可证

[MIT License](LICENSE)

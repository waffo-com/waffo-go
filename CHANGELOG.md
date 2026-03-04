# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.2] - 2026-03-03

### Fixed

- **`RefundOrderParams`**: Corrected fields to match openapi.json. Added missing `AcquiringOrderID` and `RequestedAt` (auto-injected by SDK); replaced `MerchantInfo` with top-level `MerchantID` (correctly serializes as flat `merchantId`); renamed `MerchantOrderID`→`MerchantRefundOrderID` with correct json tag; fixed `NotifyURL` json tag from `notifyUrl` to `refundNotifyUrl`; removed `PaymentRequestID` (not in API schema); added `RefundUserInfo` and `RefundUserBankInfo` types
- **E2E tests**: Added `TestRefundWebhook_FullFlow` for full payment → refund → `REFUND_NOTIFICATION` webhook validation using DANA payment

## [1.2.1] - 2026-03-01

### Fixed

- **Webhook response format**: `BuildSuccessResponse()` now returns `{"message":"success"}` instead of `{"status":"success"}`. Waffo validates both HTTP 200 and the exact response body — the previous format caused Waffo to treat all webhook notifications as failed and retry indefinitely.

## [1.2.0] - 2026-02-24

### Added

- **Timestamp auto-injection**: `OrderRequestedAt` and `RequestedAt` fields are automatically set to current UTC time when not provided by the caller
- **`CancelOrderParams.OrderRequestedAt`**: Added missing required field per openapi.json
- **`utils/time_utils.go`**: New utility for generating ISO 8601 timestamps with 3-digit milliseconds

### Fixed

- **`injectMerchantID` pointer handling**: Fixed reflection to properly handle pointer-type `MerchantInfo` fields, including nil pointer initialization

## [1.1.0] - 2026-02-12

### Added

- **Webhook E2E tests**: Real sandbox webhook notification testing via cloudflared tunnel
- **Webhook notification restructuring**: Nested `{eventType, result}` structure aligned with Java SDK
- **Complete webhook type definitions**: 75+ fields added to PaymentNotification, RefundNotification, and SubscriptionNotification result structs

### Fixed

- **CancelSubscriptionParams**: Fixed parameter structure to match openapi.json (top-level `MerchantID` instead of nested `MerchantInfo`)

## [1.0.0] - 2026-02-04

### Added

- Initial release of Waffo Go SDK
- Order management (create, inquiry, cancel, refund)
- Subscription management (create, inquiry, cancel, change, changeInquiry)
- Refund query
- Merchant configuration query
- Payment method configuration query
- Webhook handling with signature verification
- RSA signing utilities (SHA256withRSA)
- Support for custom HTTP transport
- Zero runtime dependencies (uses Go standard library only)
- E2E test infrastructure with Playwright
- E2E tests for payment flow, subscription flow, refund, and subscription change

### Features

- **Runtime**: Go 1.20+
- **HTTP Client**: `net/http` (standard library)
- **JSON**: `encoding/json` (standard library)
- **Crypto**: `crypto/rsa`, `crypto/sha256` (standard library)
- **Framework Support**: Gin, Echo, Fiber, Chi adapters planned

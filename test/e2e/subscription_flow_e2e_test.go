//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/types/subscription"
)

// SubscriptionFlowE2ETest tests the complete subscription flow:
// 1. Create subscription via SDK
// 2. Complete first payment
// 3. Query subscription status
//
// Run with:
//   go test -tags=e2e ./test/e2e/... -run TestSubscription -v

func TestSubscriptionFlow_CreateAndPay(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// 1. Create subscription
	subscriptionRequest := fmt.Sprintf("e2e_sub_%d", time.Now().UnixNano())

	params := &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    subscriptionRequest,
		MerchantSubscriptionID: fmt.Sprintf("E2E_SUB_%d", time.Now().UnixMilli()),
		Currency:               "HKD",
		Amount:                 "99.00",
		ProductInfo: &subscription.ProductInfo{
			Description:    "E2E Subscription Test Description",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
			NumberOfPeriod: "12",
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:       "e2e_test_user",
			UserEmail:    "e2e@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "E2E_GOODS_001",
			GoodsName:     "E2E Test Subscription",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/subscription/e2e",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
		RequestedAt:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:                 "https://httpbin.org/post",
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		SubscriptionManagementURL: "https://example.com/subscription/manage",
	}

	resp, err := testWaffo.Subscription().Create(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to create subscription: subscriptionRequest=%s, error=%v", subscriptionRequest, err)
	}

	t.Logf("Subscription Request: %s", subscriptionRequest)
	t.Logf("Response: code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())

	if !resp.IsSuccess() {
		t.Fatalf("Create subscription failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	if data == nil {
		t.Fatalf("Subscription data is nil: subscriptionRequest=%s, code=%s, msg=%s",
			subscriptionRequest, resp.GetCode(), resp.GetMessage())
	}

	checkoutURL := data.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: subscriptionRequest=%s, data=%+v", subscriptionRequest, data)
	}

	t.Logf("Checkout URL: %s", checkoutURL)

	// 2. Open checkout page
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to checkout: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(2000)

	base.TakeScreenshot("subscription_checkout_loaded")

	// Fill card details
	base.Fill("#payMethodProperties\\.card\\.pan", TestCard.Number)
	base.SleepMs(500)

	expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
	base.Fill("#payMethodProperties\\.card\\.expiry", expiry)
	base.SleepMs(500)

	base.Fill("#payMethodProperties\\.card\\.cvv", TestCard.CVV)
	base.SleepMs(500)

	base.Fill("#payMethodProperties\\.card\\.name", TestCard.Holder)
	base.SleepMs(500)

	base.TakeScreenshot("subscription_card_filled")

	// Submit payment
	t.Log("Submitting subscription payment...")
	if err := base.Click("button[type='submit']"); err != nil {
		t.Logf("Warning: Could not click submit button: %v", err)
	}

	// Wait for redirect
	base.SetTimeout(60000)
	if err := base.WaitForURLContains("success"); err != nil {
		t.Logf("Did not redirect to success page: %v", err)
		base.PrintCurrentURL()
	} else {
		t.Log("Subscription payment successful!")
	}

	base.TakeScreenshot("subscription_completed")

	// 3. Query subscription status
	base.SleepMs(2000)

	inquiryParams := &subscription.InquirySubscriptionParams{
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		SubscriptionRequest: subscriptionRequest,
	}

	inquiryResp, err := testWaffo.Subscription().Inquiry(context.Background(), inquiryParams, nil)
	if err != nil {
		t.Logf("Warning: Failed to query subscription: subscriptionRequest=%s, error=%v",
			subscriptionRequest, err)
		return
	}

	t.Logf("Inquiry Response: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())

	if inquiryResp.IsSuccess() {
		inquiryData := inquiryResp.GetData()
		t.Logf("Subscription Status: subscriptionRequest=%s, status=%s", subscriptionRequest, inquiryData.SubscriptionStatus)
	} else {
		t.Logf("Inquiry failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
	}
}

func TestSubscriptionFlow_QueryOnly(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	// This test only queries an existing subscription
	// Useful for checking the query API without creating new subscriptions

	subscriptionRequest := "existing_subscription_request_id"

	inquiryParams := &subscription.InquirySubscriptionParams{
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		SubscriptionRequest: subscriptionRequest,
	}

	resp, err := testWaffo.Subscription().Inquiry(context.Background(), inquiryParams, nil)
	if err != nil {
		t.Logf("Query error (expected if subscription doesn't exist): subscriptionRequest=%s, error=%v",
			subscriptionRequest, err)
		return
	}

	t.Logf("Inquiry Response: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())

	if resp.IsSuccess() {
		data := resp.GetData()
		t.Logf("Subscription Status: subscriptionRequest=%s, status=%s", subscriptionRequest, data.SubscriptionStatus)
	} else {
		t.Logf("Query failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}
}

func TestSubscriptionFlow_CancelSubscription(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// 1. Create subscription first
	subscriptionRequest := fmt.Sprintf("e2e_cancel_%d", time.Now().UnixNano())

	createParams := &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    subscriptionRequest,
		MerchantSubscriptionID: fmt.Sprintf("E2E_CANCEL_%d", time.Now().UnixMilli()),
		Currency:               "HKD",
		Amount:                 "99.00",
		ProductInfo: &subscription.ProductInfo{
			Description:    "E2E Cancel Test Description",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
			NumberOfPeriod: "12",
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:       "e2e_test_user",
			UserEmail:    "e2e@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "E2E_GOODS_002",
			GoodsName:     "E2E Cancel Test Subscription",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/subscription/cancel",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
		RequestedAt:               time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:                 "https://httpbin.org/post",
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		SubscriptionManagementURL: "https://example.com/subscription/manage",
	}

	createResp, err := testWaffo.Subscription().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create subscription: subscriptionRequest=%s, error=%v", subscriptionRequest, err)
	}

	t.Logf("Subscription Request: %s", subscriptionRequest)
	t.Logf("Response: code=%s, msg=%s, data=%+v", createResp.GetCode(), createResp.GetMessage(), createResp.GetData())

	if !createResp.IsSuccess() {
		t.Fatalf("Create subscription failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	data := createResp.GetData()
	checkoutURL := data.FetchRedirectURL()

	// 2. Complete payment
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(2000)

	// Fill card and pay
	base.Fill("#payMethodProperties\\.card\\.pan", TestCard.Number)
	base.SleepMs(300)
	expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
	base.Fill("#payMethodProperties\\.card\\.expiry", expiry)
	base.SleepMs(300)
	base.Fill("#payMethodProperties\\.card\\.cvv", TestCard.CVV)
	base.SleepMs(300)
	base.Fill("#payMethodProperties\\.card\\.name", TestCard.Holder)
	base.SleepMs(300)

	base.Click("button[type='submit']")
	base.SetTimeout(60000)
	base.WaitForURLContains("success")

	base.SleepMs(2000)

	// 3. Cancel subscription
	cancelParams := &subscription.CancelSubscriptionParams{
		SubscriptionID: data.SubscriptionID,
		MerchantID:     testConfig.MerchantID,
		RequestedAt:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	cancelResp, err := testWaffo.Subscription().Cancel(context.Background(), cancelParams, nil)
	if err != nil {
		t.Logf("Warning: Cancel failed (may be expected): subscriptionRequest=%s, subscriptionID=%s, error=%v",
			subscriptionRequest, data.SubscriptionID, err)
		return
	}

	t.Logf("Cancel Response: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, data.SubscriptionID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())

	if cancelResp.IsSuccess() {
		t.Logf("Subscription cancelled successfully: subscriptionRequest=%s, subscriptionID=%s",
			subscriptionRequest, data.SubscriptionID)
	} else {
		t.Logf("Cancel failed: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, data.SubscriptionID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())
	}

	// 4. Verify cancellation
	inquiryParams := &subscription.InquirySubscriptionParams{
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		SubscriptionRequest: subscriptionRequest,
	}

	inquiryResp, inquiryErr := testWaffo.Subscription().Inquiry(context.Background(), inquiryParams, nil)
	if inquiryErr != nil {
		t.Logf("Warning: Failed to query subscription after cancel: subscriptionRequest=%s, error=%v",
			subscriptionRequest, inquiryErr)
		return
	}

	t.Logf("Post-Cancel Inquiry Response: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())

	if inquiryResp.IsSuccess() {
		inquiryData := inquiryResp.GetData()
		t.Logf("Final Subscription Status: subscriptionRequest=%s, status=%s", subscriptionRequest, inquiryData.SubscriptionStatus)
	} else {
		t.Logf("Post-cancel inquiry failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
	}
}

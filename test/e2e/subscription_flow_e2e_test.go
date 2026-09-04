//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/v2/types/order"
	"github.com/waffo-com/waffo-go/v2/types/subscription"
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
		RequestedAt:               time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
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

	base.SetTimeout(60000)
	if !base.CompleteCheckout(t, checkoutURL, "subscription_checkout", SubscriptionCheckoutCard()) {
		t.Fatalf("Subscription checkout did not complete: subscriptionRequest=%s, subscriptionID=%s, currentURL=%s",
			subscriptionRequest, data.SubscriptionID, base.Page.URL())
	}

	inquiryData := requireSubscriptionStatus(t, subscriptionRequest, "ACTIVE", 20, 3*time.Second)
	t.Logf("Subscription Status: subscriptionRequest=%s, subscriptionID=%s, status=%s",
		subscriptionRequest, data.SubscriptionID, inquiryData.SubscriptionStatus)

	// 4. Manage subscription (get management URL)
	manageParams := &subscription.ManageSubscriptionParams{
		SubscriptionID:      data.SubscriptionID,
		SubscriptionRequest: subscriptionRequest,
	}

	manageResp, manageErr := testWaffo.Subscription().Manage(context.Background(), manageParams, nil)
	if manageErr != nil {
		t.Fatalf("Manage subscription error for ACTIVE subscription: subscriptionRequest=%s, subscriptionID=%s, error=%v",
			subscriptionRequest, data.SubscriptionID, manageErr)
	} else {
		t.Logf("Manage response: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, data.SubscriptionID, manageResp.GetCode(), manageResp.GetMessage(), manageResp.GetData())

		if manageResp.IsSuccess() {
			manageData := manageResp.GetData()
			if manageData == nil || manageData.ManagementURL == "" {
				t.Fatalf("Manage did not return management URL for ACTIVE subscription: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
					subscriptionRequest, data.SubscriptionID, manageResp.GetCode(), manageResp.GetMessage(), manageResp.GetData())
			}
			t.Logf("Management URL: %s", manageData.ManagementURL)
		} else {
			t.Fatalf("Manage failed for ACTIVE subscription: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
				subscriptionRequest, data.SubscriptionID, manageResp.GetCode(), manageResp.GetMessage(), manageResp.GetData())
		}
	}
}

func TestSubscriptionFlow_UpdateWithTopup(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	now := time.Now()
	subscriptionRequest := fmt.Sprintf("e2e_topup_%d", now.UnixNano())
	createParams := &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    subscriptionRequest,
		MerchantSubscriptionID: fmt.Sprintf("E2E_TOPUP_%d", now.UnixMilli()),
		Currency:               "USD",
		Amount:                 "99.00",
		ProductInfo: &subscription.ProductInfo{
			Description:    "E2E subscription top-up",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
			NumberOfPeriod: "12",
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:       "e2e_topup_user",
			UserEmail:    "e2e-topup@waffo.co",
			UserTerminal: "WEB",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "E2E_TOPUP_GOODS",
			GoodsName:     "E2E Top-up Subscription",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/subscription/topup",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
		RequestedAt:               now.UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:                 "https://httpbin.org/post",
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		SubscriptionManagementURL: "https://example.com/subscription/manage",
	}

	createResp, err := testWaffo.Subscription().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Create subscription failed: subscriptionRequest=%s, error=%v", subscriptionRequest, err)
	}
	if !createResp.IsSuccess() || createResp.GetData() == nil {
		t.Fatalf("Create subscription failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	defer func() {
		cancelResp, cancelErr := testWaffo.Subscription().Cancel(context.Background(), &subscription.CancelSubscriptionParams{
			SubscriptionID: createData.SubscriptionID,
			MerchantID:     testConfig.MerchantID,
			RequestedAt:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		}, nil)
		if cancelErr != nil || !cancelResp.IsSuccess() {
			t.Logf("Subscription cleanup did not complete: subscriptionRequest=%s, subscriptionID=%s, error=%v, response=%+v",
				subscriptionRequest, createData.SubscriptionID, cancelErr, cancelResp)
			return
		}
		t.Logf("Subscription cleanup completed: subscriptionRequest=%s, subscriptionID=%s",
			subscriptionRequest, createData.SubscriptionID)
	}()

	checkoutURL := createData.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Subscription checkout URL is empty: subscriptionRequest=%s, subscriptionID=%s",
			subscriptionRequest, createData.SubscriptionID)
	}
	base.SetTimeout(60000)
	if !base.CompleteCheckout(t, checkoutURL, "subscription_topup_activation", SubscriptionCheckoutCard()) {
		t.Fatalf("Subscription checkout did not complete: subscriptionRequest=%s, subscriptionID=%s, currentURL=%s",
			subscriptionRequest, createData.SubscriptionID, base.Page.URL())
	}
	requireSubscriptionStatus(t, subscriptionRequest, "ACTIVE", 20, 3*time.Second)

	topupRequest := fmt.Sprintf("e2e-topup-%d", time.Now().UnixMilli())
	updateResp, err := testWaffo.Subscription().Update(context.Background(), &subscription.UpdateSubscriptionParams{
		SubscriptionID: createData.SubscriptionID,
		Amount:         "99.00",
		TopupInfo: &subscription.SubscriptionUpdateTopupInfo{
			TopupRequest:   topupRequest,
			TopupAmount:    "1.00",
			Description:    "E2E seat add-on",
			OrderExpiredAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		},
	}, nil)
	if err != nil {
		t.Fatalf("Top-up update failed: subscriptionRequest=%s, subscriptionID=%s, topupRequest=%s, error=%v",
			subscriptionRequest, createData.SubscriptionID, topupRequest, err)
	}
	if !updateResp.IsSuccess() || updateResp.GetData() == nil || updateResp.GetData().TopupInfo == nil {
		t.Fatalf("Top-up update failed: subscriptionRequest=%s, subscriptionID=%s, topupRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, createData.SubscriptionID, topupRequest,
			updateResp.GetCode(), updateResp.GetMessage(), updateResp.GetData())
	}

	updateData := updateResp.GetData()
	topupData := updateData.TopupInfo
	if updateData.SubscriptionID != createData.SubscriptionID || topupData.TopupRequest != topupRequest ||
		topupData.TopupAmount == "" || topupData.AcquiringOrderID == "" || topupData.TopupStatus == "" {
		t.Fatalf("Unexpected top-up response: subscriptionRequest=%s, subscriptionID=%s, topupRequest=%s, data=%+v",
			subscriptionRequest, createData.SubscriptionID, topupRequest, updateData)
	}
	t.Logf("Top-up accepted: subscriptionRequest=%s, subscriptionID=%s, topupRequest=%s, acquiringOrderID=%s, amount=%s, status=%s",
		subscriptionRequest, createData.SubscriptionID, topupRequest,
		topupData.AcquiringOrderID, topupData.TopupAmount, topupData.TopupStatus)

	if topupData.TopupStatus == "AUTHORIZATION_REQUIRED" {
		topupURL := (&order.CreateOrderData{OrderAction: topupData.TopupAction}).FetchRedirectURL()
		if topupURL == "" {
			t.Fatalf("Top-up authorization URL is empty: acquiringOrderID=%s, action=%s",
				topupData.AcquiringOrderID, topupData.TopupAction)
		}
		if !base.CompleteCheckout(t, topupURL, "subscription_topup_checkout", SubscriptionCheckoutCard()) {
			t.Fatalf("Top-up checkout did not complete: acquiringOrderID=%s, currentURL=%s",
				topupData.AcquiringOrderID, base.Page.URL())
		}
	}

	var finalOrderStatus string
	var lastInquiry string
	for attempt := 1; attempt <= 20; attempt++ {
		inquiryResp, inquiryErr := testWaffo.Order().Inquiry(context.Background(), &order.InquiryOrderParams{
			AcquiringOrderID: topupData.AcquiringOrderID,
		}, nil)
		if inquiryErr != nil {
			lastInquiry = inquiryErr.Error()
		} else {
			lastInquiry = fmt.Sprintf("code=%s, msg=%s, data=%+v",
				inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
			if inquiryResp.IsSuccess() && inquiryResp.GetData() != nil {
				finalOrderStatus = inquiryResp.GetData().OrderStatus
				if finalOrderStatus == "PAY_SUCCESS" {
					break
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
	if finalOrderStatus != "PAY_SUCCESS" {
		t.Fatalf("Top-up order did not reach PAY_SUCCESS: acquiringOrderID=%s, finalStatus=%s, lastInquiry=%s",
			topupData.AcquiringOrderID, finalOrderStatus, lastInquiry)
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
		SubscriptionRequest: subscriptionRequest,
	}

	resp, err := testWaffo.Subscription().Inquiry(context.Background(), inquiryParams, nil)
	if err != nil {
		t.Logf("Query error (expected if subscription doesn't exist): subscriptionRequest=%s, error=%v",
			subscriptionRequest, err)
		t.Skip("Placeholder subscription is not configured in sandbox")
	}

	t.Logf("Inquiry Response: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())

	if resp.IsSuccess() {
		data := resp.GetData()
		t.Logf("Subscription Status: subscriptionRequest=%s, status=%s", subscriptionRequest, data.SubscriptionStatus)
	} else {
		t.Logf("Query failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())
		t.Skipf("Placeholder subscription is not available: code=%s, msg=%s", resp.GetCode(), resp.GetMessage())
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
	if data == nil {
		t.Fatalf("Subscription data is nil before cancel: subscriptionRequest=%s, code=%s, msg=%s",
			subscriptionRequest, createResp.GetCode(), createResp.GetMessage())
	}
	checkoutURL := data.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty before cancel: subscriptionRequest=%s, subscriptionID=%s, data=%+v",
			subscriptionRequest, data.SubscriptionID, data)
	}

	base.SetTimeout(60000)
	if !base.CompleteCheckout(t, checkoutURL, "subscription_cancel_checkout", SubscriptionCheckoutCard()) {
		t.Fatalf("Subscription checkout did not complete before cancel: subscriptionRequest=%s, subscriptionID=%s, currentURL=%s",
			subscriptionRequest, data.SubscriptionID, base.Page.URL())
	}

	// 3. Verify subscription is ACTIVE before attempting cancellation.
	inquiryData := requireSubscriptionStatus(t, subscriptionRequest, "ACTIVE", 20, 3*time.Second)
	t.Logf("Subscription is ACTIVE before cancel: subscriptionRequest=%s, subscriptionID=%s, status=%s",
		subscriptionRequest, data.SubscriptionID, inquiryData.SubscriptionStatus)

	// 4. Cancel subscription
	cancelParams := &subscription.CancelSubscriptionParams{
		SubscriptionID: data.SubscriptionID,
		MerchantID:     testConfig.MerchantID,
		RequestedAt:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	cancelResp, err := testWaffo.Subscription().Cancel(context.Background(), cancelParams, nil)
	if err != nil {
		t.Fatalf("Cancel failed for ACTIVE subscription: subscriptionRequest=%s, subscriptionID=%s, error=%v",
			subscriptionRequest, data.SubscriptionID, err)
	}

	t.Logf("Cancel Response: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, data.SubscriptionID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())

	if cancelResp.IsSuccess() {
		t.Logf("Subscription cancelled successfully: subscriptionRequest=%s, subscriptionID=%s",
			subscriptionRequest, data.SubscriptionID)
	} else {
		t.Fatalf("Cancel failed for ACTIVE subscription: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, data.SubscriptionID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())
	}

	// 5. Verify cancellation.
	var finalStatus string
	inquiryParams := &subscription.InquirySubscriptionParams{
		SubscriptionRequest: subscriptionRequest,
	}
	for attempt := 1; attempt <= 6; attempt++ {
		inquiryResp, inquiryErr := testWaffo.Subscription().Inquiry(context.Background(), inquiryParams, nil)
		if inquiryErr != nil {
			t.Fatalf("Failed to query subscription after cancel: subscriptionRequest=%s, error=%v",
				subscriptionRequest, inquiryErr)
		}
		t.Logf("Post-Cancel Inquiry Response: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
		if !inquiryResp.IsSuccess() {
			t.Fatalf("Post-cancel inquiry failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
				subscriptionRequest, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
		}
		inquiryData = inquiryResp.GetData()
		if inquiryData == nil {
			t.Fatalf("Post-cancel inquiry data is nil: subscriptionRequest=%s", subscriptionRequest)
		}
		finalStatus = inquiryData.SubscriptionStatus
		if finalStatus == "MERCHANT_CANCELLED" {
			t.Logf("Final Subscription Status: subscriptionRequest=%s, status=%s", subscriptionRequest, finalStatus)
			return
		}
		t.Logf("Waiting for cancellation status: subscriptionRequest=%s, attempt=%d, status=%s",
			subscriptionRequest, attempt, finalStatus)
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("Subscription did not reach MERCHANT_CANCELLED after cancel: subscriptionRequest=%s, finalStatus=%s",
		subscriptionRequest, finalStatus)
}

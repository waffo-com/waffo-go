//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/waffo-com/waffo-go/v2/types/subscription"
)

// SubscriptionChangeE2ETest tests the subscription change (upgrade/downgrade) flow:
// 1. Create original subscription
// 2. Complete first payment (activate subscription)
// 3. Execute subscription change (upgrade)
// 4. Query change status with ChangeInquiry()
//
// Run with:
//   go test -tags=e2e ./test/e2e/... -run TestSubscriptionChange -v
//
// Visual mode (non-headless):
//   E2E_HEADLESS=false go test -tags=e2e ./test/e2e/... -run TestSubscriptionChange -v

func TestSubscriptionChange_UpgradeSubscription(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// Generate unique IDs for this test
	userId := fmt.Sprintf("e2e_change_user_%d", time.Now().UnixNano())
	originalSubscriptionRequest := fmt.Sprintf("e2e_orig_sub_%d", time.Now().UnixNano())

	// ==================== Step 1: Create Original Subscription ====================
	t.Log("=== Step 1: Creating original subscription ===")

	createParams := &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    originalSubscriptionRequest,
		MerchantSubscriptionID: fmt.Sprintf("E2E_ORIG_%d", time.Now().UnixMilli()),
		Currency:               "HKD",
		Amount:                 "99.00",
		ProductInfo: &subscription.ProductInfo{
			Description:    "E2E Original Monthly Subscription",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
			NumberOfPeriod: "12",
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:       userId,
			UserEmail:    userId + "@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "E2E_GOODS_ORIG",
			GoodsName:     "E2E Original Subscription",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/subscription/original",
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
		t.Fatalf("Failed to create original subscription: subscriptionRequest=%s, error=%v",
			originalSubscriptionRequest, err)
	}

	t.Logf("Original Subscription Request: %s", originalSubscriptionRequest)
	t.Logf("Response: code=%s, msg=%s, data=%+v", createResp.GetCode(), createResp.GetMessage(), createResp.GetData())

	if !createResp.IsSuccess() {
		t.Fatalf("Create original subscription failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			originalSubscriptionRequest, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("Original subscription data is nil: subscriptionRequest=%s, code=%s, msg=%s",
			originalSubscriptionRequest, createResp.GetCode(), createResp.GetMessage())
	}

	originalSubscriptionID := createData.SubscriptionID
	t.Logf("Original Subscription ID: %s", originalSubscriptionID)

	// ==================== Step 2: Activate Original Subscription ====================
	t.Log("=== Step 2: Activating original subscription ===")

	checkoutURL := createData.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Original subscription checkout URL is empty: subscriptionRequest=%s, subscriptionID=%s, data=%+v",
			originalSubscriptionRequest, originalSubscriptionID, createData)
	} else {
		base.SetTimeout(60000)
		if !base.CompleteCheckout(t, checkoutURL, "subscription_change_original_checkout", SubscriptionCheckoutCard()) {
			t.Fatalf("Original subscription checkout did not complete: subscriptionRequest=%s, subscriptionID=%s, currentURL=%s",
				originalSubscriptionRequest, originalSubscriptionID, base.Page.URL())
		}
	}

	inquiryData := requireSubscriptionStatus(t, originalSubscriptionRequest, "ACTIVE", 20, 3*time.Second)
	t.Logf("Original Subscription Status: subscriptionRequest=%s, subscriptionID=%s, status=%s",
		originalSubscriptionRequest, originalSubscriptionID, inquiryData.SubscriptionStatus)

	// ==================== Step 3: Execute Subscription Change (Upgrade) ====================
	t.Log("=== Step 3: Executing subscription change (upgrade) ===")

	newSubscriptionRequest := fmt.Sprintf("e2e_new_sub_%d", time.Now().UnixNano())
	orderExpiredAt := time.Now().Add(30 * time.Minute).UTC().Format("2006-01-02T15:04:05.000Z")

	changeParams := &subscription.ChangeSubscriptionParams{
		SubscriptionRequest:       newSubscriptionRequest,
		MerchantSubscriptionID:    fmt.Sprintf("E2E_CHANGE_%d", time.Now().UnixMilli()),
		OriginSubscriptionRequest: originalSubscriptionRequest,
		RemainingAmount:           "50.00",
		Currency:                  "HKD",
		RequestedAt:               time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		OrderExpiredAt:            orderExpiredAt,
		NotifyURL:                 "https://httpbin.org/post",
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		SubscriptionManagementURL: "https://example.com/subscription/manage",
		ProductInfoList: []subscription.SubscriptionChangeProductInfo{
			{
				Description:    "E2E Premium Monthly Subscription (Upgraded)",
				PeriodType:     "MONTHLY",
				PeriodInterval: "1",
				Amount:         "199.00",
				NumberOfPeriod: "12",
			},
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:    userId,
			UserEmail: userId + "@test.com",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "E2E_GOODS_PREMIUM",
			GoodsName:     "E2E Premium Subscription",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/subscription/premium",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
	}

	t.Logf("New Subscription Request: %s", newSubscriptionRequest)
	t.Logf("Origin Subscription Request: %s", originalSubscriptionRequest)

	changeResp, err := testWaffo.Subscription().Change(context.Background(), changeParams, nil)
	if err != nil {
		t.Fatalf("Failed to change subscription: newSubscriptionRequest=%s, originSubscriptionRequest=%s, error=%v",
			newSubscriptionRequest, originalSubscriptionRequest, err)
	}

	t.Logf("Change Response: code=%s, msg=%s, data=%+v", changeResp.GetCode(), changeResp.GetMessage(), changeResp.GetData())

	if !changeResp.IsSuccess() {
		t.Fatalf("Change failed after ACTIVE prerequisite: newSubscriptionRequest=%s, originSubscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			newSubscriptionRequest, originalSubscriptionRequest, changeResp.GetCode(), changeResp.GetMessage(), changeResp.GetData())
	}

	changeData := changeResp.GetData()
	if changeData == nil {
		t.Fatalf("Change subscription data is nil: newSubscriptionRequest=%s, code=%s, msg=%s",
			newSubscriptionRequest, changeResp.GetCode(), changeResp.GetMessage())
	}

	t.Log("Subscription change initiated successfully!")
	t.Logf("New Subscription ID: %s", changeData.SubscriptionID)
	t.Logf("Change Status: %s", changeData.SubscriptionChangeStatus)

	base.TakeScreenshot("subscription_change_step3_completed")

	if changeData.SubscriptionChangeStatus == "AUTHORIZATION_REQUIRED" {
		authURL := changeData.FetchRedirectURL()
		if authURL == "" {
			t.Fatalf("Subscription change requires authorization but no auth URL returned: newSubscriptionRequest=%s, originSubscriptionRequest=%s, data=%+v",
				newSubscriptionRequest, originalSubscriptionRequest, changeData)
		}
		base.SetTimeout(60000)
		if !base.CompleteCheckout(t, authURL, "subscription_change_auth_checkout", SubscriptionCheckoutCard()) {
			t.Fatalf("Subscription change authorization did not complete: newSubscriptionRequest=%s, originSubscriptionRequest=%s, currentURL=%s",
				newSubscriptionRequest, originalSubscriptionRequest, base.Page.URL())
		}
	}

	// ==================== Step 4: Query Change Status ====================
	t.Log("=== Step 4: Querying subscription change status ===")
	changeInquiryData := requireSubscriptionChangeStatus(t, originalSubscriptionRequest, newSubscriptionRequest, "SUCCESS", 20, 3*time.Second)
	t.Logf("Change Inquiry: subscriptionID=%s, changeStatus=%s, originSubscriptionRequest=%s",
		changeInquiryData.SubscriptionID, changeInquiryData.SubscriptionChangeStatus, changeInquiryData.OriginSubscriptionRequest)

	base.TakeScreenshot("subscription_change_step4_inquiry")
	t.Log("=== Subscription Change Test Completed ===")
}

func TestSubscriptionChange_QueryOnly(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	// This test only queries an existing subscription change
	// Useful for checking the query API without creating new subscriptions

	subscriptionRequest := "existing_change_request_id"
	originSubscriptionRequest := "existing_origin_request_id"

	changeInquiryParams := &subscription.ChangeInquiryParams{
		OriginSubscriptionRequest: originSubscriptionRequest,
		SubscriptionRequest:       subscriptionRequest,
	}

	resp, err := testWaffo.Subscription().ChangeInquiry(context.Background(), changeInquiryParams, nil)
	if err != nil {
		t.Logf("Query error (expected if change doesn't exist): subscriptionRequest=%s, error=%v",
			subscriptionRequest, err)
		t.Skip("Placeholder subscription change is not configured in sandbox")
	}

	t.Logf("Change Inquiry Response: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
		subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())

	if resp.IsSuccess() {
		data := resp.GetData()
		if data != nil {
			t.Logf("Change Status: subscriptionRequest=%s, changeStatus=%s",
				subscriptionRequest, data.SubscriptionChangeStatus)
		}
	} else {
		t.Logf("Query failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			subscriptionRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())
		t.Skipf("Placeholder subscription change is not available: code=%s, msg=%s", resp.GetCode(), resp.GetMessage())
	}
}

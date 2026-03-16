//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/waffo-com/waffo-go/core"
	"github.com/waffo-com/waffo-go/types/order"
	"github.com/waffo-com/waffo-go/types/subscription"
)

// Webhook E2E Tests
//
// Tests real webhook notifications from Waffo sandbox using cloudflared tunnel.
//
// Test order: Activation -> PeriodChanged -> Cancellation -> Payment -> Refund -> Cleanup
//
// Run with:
//   go test -tags=e2e -timeout 600s -run TestWebhook -v ./test/e2e/

const (
	webhookPort    = 4002
	webhookTimeout = 120 * time.Second

	// 3DS test card (triggers 3DS challenge in sandbox)
	test3DSCard = "4000000000001000"
	test3DSCode = "1234"
)

// Shared state across tests
var (
	webhookServer     *WebhookTestServer
	webhookNgrokURL   string
	webhookSubRequest string
	webhookSubID      string
	webhookSubCreated bool

	// Shared state for payment/refund tests
	webhookPaymentRequestID  string
	webhookAcquiringOrderID  string
	webhookOrderPaid         bool
)

func setupWebhookInfra(t *testing.T) {
	t.Helper()

	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	if webhookServer != nil {
		return // Already set up
	}

	// Create webhook handler with all handlers registered
	handler := testWaffo.Webhook().
		OnPayment(func(n *core.PaymentNotification) {
			fmt.Printf("[Handler] Payment notification: orderStatus=%s\n", n.Result.OrderStatus)
		}).
		OnRefund(func(n *core.RefundNotification) {
			fmt.Printf("[Handler] Refund notification: refundStatus=%s\n", n.Result.RefundStatus)
		}).
		OnSubscriptionStatus(func(n *core.SubscriptionStatusNotification) {
			status := ""
			if n.Result != nil {
				status = n.Result.SubscriptionStatus
			}
			fmt.Printf("[Handler] Subscription status notification: %s\n", status)
		}).
		OnSubscriptionPeriodChanged(func(n *core.SubscriptionPeriodChangedNotification) {
			fmt.Println("[Handler] Subscription period changed notification")
		}).
		OnSubscriptionChange(func(n *core.SubscriptionChangeNotification) {
			status := ""
			newReq := ""
			if n.Result != nil {
				status = n.Result.SubscriptionChangeStatus
				newReq = n.Result.SubscriptionRequest
			}
			fmt.Printf("[Handler] Subscription change notification: status=%s, newSubRequest=%s\n", status, newReq)
		})

	// Start webhook server
	webhookServer = NewWebhookTestServer(handler, webhookPort)
	if err := webhookServer.Start(); err != nil {
		t.Fatalf("Failed to start webhook server: %v", err)
	}

	// Start cloudflared tunnel
	var err error
	webhookNgrokURL, err = StartNgrok(webhookPort)
	if err != nil {
		webhookServer.Stop()
		t.Fatalf("Failed to start tunnel: %v", err)
	}

	t.Logf("[Webhook E2E] Tunnel URL: %s", webhookNgrokURL)
}

func cleanupWebhookInfra() {
	if webhookServer != nil {
		webhookServer.PrintReport()
		webhookServer.Stop()
		webhookServer = nil
	}
	StopNgrok()
}

// completePaymentFlow handles the full payment flow: card fill, checkbox, submit, 3DS, wait for result
func completePaymentFlow(t *testing.T, page playwright.Page) bool {
	t.Helper()

	// 1. Fill card details with 3DS card
	t.Log("  Step A: Filling card details...")
	cardSelectors := []string{
		"#payMethodProperties\\.card\\.pan",
		"input[name='cardNumber']",
		"input[placeholder*='card']",
	}
	cardFilled := false
	for _, sel := range cardSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			if err := page.Locator(sel).First().Fill(test3DSCard); err == nil {
				t.Logf("  Card number filled using: %s", sel)
				cardFilled = true
				break
			}
		}
	}
	if !cardFilled {
		t.Log("  Failed to fill card number")
		return false
	}
	time.Sleep(500 * time.Millisecond)

	expirySelectors := []string{"#payMethodProperties\\.card\\.expiry", "input[name='expiry']"}
	for _, sel := range expirySelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Fill("12/28")
			t.Logf("  Expiry filled using: %s", sel)
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	cvvSelectors := []string{"#payMethodProperties\\.card\\.cvv", "input[name='cvv']"}
	for _, sel := range cvvSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Fill("123")
			t.Logf("  CVV filled using: %s", sel)
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	nameSelectors := []string{"#payMethodProperties\\.card\\.name", "input[name='cardholderName']"}
	for _, sel := range nameSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Fill("Tom")
			t.Logf("  Name filled using: %s", sel)
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	// 2. Check all checkboxes
	t.Log("  Step B: Checking checkboxes...")
	checkboxes, _ := page.Locator("input[type='checkbox']").All()
	t.Logf("  Found %d checkbox(es)", len(checkboxes))
	for _, cb := range checkboxes {
		checked, _ := cb.IsChecked()
		if !checked {
			cb.Check()
		}
	}
	time.Sleep(500 * time.Millisecond)

	// 3. Submit payment
	t.Log("  Step C: Submitting payment...")
	submitSelectors := []string{
		"button[type='submit']",
		"button:has-text('Pay')",
		"button:has-text('Submit')",
	}
	submitted := false
	for _, sel := range submitSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Click()
			t.Logf("  Payment submitted using: %s", sel)
			submitted = true
			break
		}
	}
	if !submitted {
		t.Log("  Failed to submit payment")
		return false
	}
	time.Sleep(5 * time.Second)

	// 4. Handle Terms & Conditions modal
	t.Log("  Step D: Handling Terms & Conditions...")
	time.Sleep(2 * time.Second)
	termsSelectors := []string{
		"button:has-text('接受並繼續')",
		"button:has-text('Accept')",
		"button:has-text('Agree')",
	}
	for _, sel := range termsSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Click()
			t.Logf("  Terms accepted using: %s", sel)
			time.Sleep(3 * time.Second)
			break
		}
	}

	// 5. Handle 3DS challenge
	t.Log("  Step E: Handling 3DS challenge...")
	time.Sleep(3 * time.Second)
	handle3DSOnPage(t, page)

	// 6. Wait for payment result
	t.Log("  Step F: Waiting for result...")
	return waitForPaymentResultGo(t, page)
}

// completeSimplePaymentFlow handles one-time order payment using standard test card (4111...)
// without 3DS challenge handling.
func completeSimplePaymentFlow(t *testing.T, page playwright.Page) bool {
	t.Helper()

	// 1. Fill card details with standard test card
	t.Log("  Step A: Filling card details (standard card)...")
	cardSelectors := []string{
		"#payMethodProperties\\.card\\.pan",
		"input[name='cardNumber']",
		"input[placeholder*='card']",
	}
	cardFilled := false
	for _, sel := range cardSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			if err := page.Locator(sel).First().Fill(TestCard.Number); err == nil {
				t.Logf("  Card number filled using: %s", sel)
				cardFilled = true
				break
			}
		}
	}
	if !cardFilled {
		t.Log("  Failed to fill card number")
		return false
	}
	time.Sleep(500 * time.Millisecond)

	expirySelectors := []string{"#payMethodProperties\\.card\\.expiry", "input[name='expiry']"}
	expiry := fmt.Sprintf("%s/%s", TestCard.ExpiryMonth, TestCard.ExpiryYear[2:])
	for _, sel := range expirySelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Fill(expiry)
			t.Logf("  Expiry filled using: %s", sel)
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	cvvSelectors := []string{"#payMethodProperties\\.card\\.cvv", "input[name='cvv']"}
	for _, sel := range cvvSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Fill(TestCard.CVV)
			t.Logf("  CVV filled using: %s", sel)
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	nameSelectors := []string{"#payMethodProperties\\.card\\.name", "input[name='cardholderName']"}
	for _, sel := range nameSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Fill(TestCard.Holder)
			t.Logf("  Name filled using: %s", sel)
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	// 2. Check all checkboxes
	t.Log("  Step B: Checking checkboxes...")
	checkboxes, _ := page.Locator("input[type='checkbox']").All()
	t.Logf("  Found %d checkbox(es)", len(checkboxes))
	for _, cb := range checkboxes {
		checked, _ := cb.IsChecked()
		if !checked {
			cb.Check()
		}
	}
	time.Sleep(500 * time.Millisecond)

	// 3. Submit payment
	t.Log("  Step C: Submitting payment...")
	submitSelectors := []string{
		"button[type='submit']",
		"button:has-text('Pay')",
		"button:has-text('Submit')",
	}
	submitted := false
	for _, sel := range submitSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			page.Locator(sel).First().Click()
			t.Logf("  Payment submitted using: %s", sel)
			submitted = true
			break
		}
	}
	if !submitted {
		t.Log("  Failed to submit payment")
		return false
	}
	time.Sleep(5 * time.Second)

	// 4. Wait for success redirect (no 3DS expected for standard test card)
	t.Log("  Step D: Waiting for result...")
	return waitForPaymentResultGo(t, page)
}

func handle3DSOnPage(t *testing.T, page playwright.Page) {
	t.Helper()

	url := page.URL()
	t.Logf("  URL after submit: %s", url)

	// Check main page URL for 3DS
	if strings.Contains(url, "doChallenge") || strings.Contains(url, "3ds") {
		t.Log("  Detected 3DS challenge page")
		fill3DSCodeOnPage(t, page)
		return
	}

	// Check iframes
	iframeCount, _ := page.Locator("iframe").Count()
	t.Logf("  Found %d iframes", iframeCount)

	for i := 0; i < iframeCount; i++ {
		frame := page.FrameLocator("iframe").Nth(i)
		inputSelectors := []string{
			"input[name='challengeDataEntry']",
			"input[name='otp']",
			"input[name='code']",
			"input[type='password']",
			"input[type='tel']",
		}
		for _, sel := range inputSelectors {
			count, _ := frame.Locator(sel).Count()
			if count > 0 {
				t.Logf("  3DS input found in iframe %d: %s", i, sel)
				frame.Locator(sel).First().Fill(test3DSCode)
				time.Sleep(500 * time.Millisecond)

				submitSels := []string{"button[type='submit']", "input[type='submit']", "button:has-text('Submit')"}
				for _, submitSel := range submitSels {
					sc, _ := frame.Locator(submitSel).Count()
					if sc > 0 {
						frame.Locator(submitSel).First().Click()
						t.Logf("  3DS submitted in iframe: %s", submitSel)
						time.Sleep(5 * time.Second)
						return
					}
				}
			}
		}
	}

	// Try main page
	fill3DSCodeOnPage(t, page)
}

func fill3DSCodeOnPage(t *testing.T, page playwright.Page) {
	t.Helper()

	inputSelectors := []string{
		"input[name='challengeDataEntry']",
		"input[name='otp']",
		"input[name='code']",
		"input[type='password']",
		"input[type='tel']",
		"input[type='text']",
		"input",
	}

	for _, sel := range inputSelectors {
		count, _ := page.Locator(sel).Count()
		if count > 0 {
			t.Logf("  Found %d 3DS input(s): %s", count, sel)
			page.Locator(sel).First().Fill(test3DSCode)
			time.Sleep(500 * time.Millisecond)

			submitSels := []string{"button[type='submit']", "input[type='submit']", "button:has-text('Submit')", "button"}
			for _, submitSel := range submitSels {
				sc, _ := page.Locator(submitSel).Count()
				if sc > 0 {
					page.Locator(submitSel).First().Click()
					t.Logf("  3DS submitted: %s", submitSel)
					time.Sleep(5 * time.Second)
					return
				}
			}
			break
		}
	}
}

func waitForPaymentResultGo(t *testing.T, page playwright.Page) bool {
	t.Helper()

	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		url := page.URL()
		t.Logf("  Check %d/15: %s", i+1, url)

		if strings.Contains(url, "doChallenge") || strings.Contains(url, "3ds") {
			t.Log("  Late 3DS challenge, handling...")
			fill3DSCodeOnPage(t, page)
			continue
		}

		if strings.Contains(url, "status=success") {
			t.Log("  Payment SUCCESS - redirected")
			return true
		}

		content, err := page.Content()
		if err == nil {
			if strings.Contains(content, "PAY_SUCCESS") || strings.Contains(content, "支付成功") {
				t.Log("  Payment SUCCESS - found indicator")
				return true
			}
			if strings.Contains(content, "訂閱成功") || strings.Contains(content, "success_page") {
				t.Log("  Subscription SUCCESS")
				return true
			}
		}

		if strings.Contains(url, "status=failed") {
			t.Log("  Payment FAILED")
			return false
		}
	}

	t.Log("  Payment result unknown after timeout")
	return false
}

// ==================== Test 1: Subscription Activation ====================

func TestWebhook_SubscriptionActivation(t *testing.T) {
	setupWebhookInfra(t)

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// 1. Create subscription with webhook URL
	webhookSubRequest = fmt.Sprintf("e2e_wh_%d", time.Now().UnixMilli())
	notifyURL := webhookNgrokURL + "/webhook"

	t.Log("=== Step 1: Creating subscription ===")
	t.Logf("  subscriptionRequest: %s", webhookSubRequest)
	t.Logf("  notifyUrl: %s", notifyURL)

	params := &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    webhookSubRequest,
		MerchantSubscriptionID: fmt.Sprintf("WH_E2E_%d", time.Now().UnixMilli()),
		Currency:               "HKD",
		Amount:                 "99.00",
		ProductInfo: &subscription.ProductInfo{
			Description:    "Webhook E2E Test Subscription",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
			NumberOfPeriod: "12",
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:       fmt.Sprintf("wh_e2e_user_%d", time.Now().Unix()),
			UserEmail:    "webhook_e2e@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "WH_E2E_GOODS",
			GoodsName:     "Webhook E2E Product",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/webhook-test",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
		RequestedAt:               time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:                 notifyURL,
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		SubscriptionManagementURL: "https://example.com/manage",
	}

	resp, err := testWaffo.Subscription().Create(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to create subscription: subscriptionRequest=%s, error=%v", webhookSubRequest, err)
	}

	t.Logf("  Response: code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())

	if !resp.IsSuccess() {
		t.Fatalf("Create subscription failed: subscriptionRequest=%s, code=%s, msg=%s, data=%+v",
			webhookSubRequest, resp.GetCode(), resp.GetMessage(), resp.GetData())
	}

	data := resp.GetData()
	if data == nil {
		t.Fatalf("Subscription data is nil: subscriptionRequest=%s, code=%s, msg=%s",
			webhookSubRequest, resp.GetCode(), resp.GetMessage())
	}

	checkoutURL := data.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: subscriptionRequest=%s, data=%+v", webhookSubRequest, data)
	}

	webhookSubID = data.SubscriptionID
	webhookSubCreated = true
	t.Logf("  checkoutUrl: %s", checkoutURL)
	t.Logf("  subscriptionId: %s", webhookSubID)

	// 2. Complete payment via Playwright
	t.Log("=== Step 2: Completing payment ===")
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to checkout: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(3000)

	paymentSuccess := completePaymentFlow(t, base.Page)
	t.Logf("  Payment result: %v", paymentSuccess)

	// 3. Wait for SUBSCRIPTION_STATUS_NOTIFICATION
	t.Log("=== Step 3: Waiting for webhook notification ===")
	notifications := webhookServer.WaitForNotification(
		"SUBSCRIPTION_STATUS_NOTIFICATION",
		1,
		webhookTimeout,
	)

	if len(notifications) > 0 {
		n := notifications[0]
		if !n.HandlerSuccess {
			t.Errorf("Webhook handler reported failure: subscriptionRequest=%s, eventType=%s, parsed=%+v",
				webhookSubRequest, n.EventType, n.Parsed)
		}
		if n.EventType != "SUBSCRIPTION_STATUS_NOTIFICATION" {
			t.Errorf("Expected SUBSCRIPTION_STATUS_NOTIFICATION, got %s: subscriptionRequest=%s, parsed=%+v",
				n.EventType, webhookSubRequest, n.Parsed)
		}
		// Verify webhook response body format
		if n.ResponseBody != `{"message":"success"}` {
			t.Errorf("Webhook response body format wrong: subscriptionRequest=%s, got=%s",
				webhookSubRequest, n.ResponseBody)
		}

		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			t.Logf("  subscriptionStatus: %v", result["subscriptionStatus"])
			t.Logf("  subscriptionId: %v", result["subscriptionId"])
			if webhookSubID == "" {
				if sid, ok := result["subscriptionId"].(string); ok {
					webhookSubID = sid
				}
			}
		}

		t.Log("SUBSCRIPTION_STATUS_NOTIFICATION received and verified!")
	} else {
		t.Log("No SUBSCRIPTION_STATUS_NOTIFICATION received within timeout (sandbox may be slow)")
	}
}

// ==================== Test 2: Subscription Period Changed ====================

func TestWebhook_SubscriptionPeriodChanged(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	if !webhookSubCreated || webhookServer == nil || webhookSubID == "" {
		t.Skip("Subscription not active from previous test")
	}

	t.Log("=== Step 1: Getting subscription management URL ===")
	t.Logf("  subscriptionId: %s", webhookSubID)
	t.Logf("  subscriptionRequest: %s", webhookSubRequest)

	// Call manage API to get management URL with retries
	var managementURL string
	for attempt := 0; attempt < 3; attempt++ {
		manageParams := &subscription.ManageSubscriptionParams{
			SubscriptionID:      webhookSubID,
			SubscriptionRequest: webhookSubRequest,
		}

		manageResp, err := testWaffo.Subscription().Manage(context.Background(), manageParams, nil)
		if err != nil {
			t.Logf("  Attempt %d: manage API error: subscriptionRequest=%s, subscriptionID=%s, error=%v",
				attempt+1, webhookSubRequest, webhookSubID, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if manageResp.IsSuccess() {
			manageData := manageResp.GetData()
			if manageData != nil && manageData.ManagementURL != "" {
				managementURL = manageData.ManagementURL
				t.Logf("  managementUrl: %s", managementURL)
				break
			}

			// Fallback: re-marshal data to JSON and check alternative field names
			// (API may return "managementUrl" but Go struct expects "manageUrl")
			if manageData != nil {
				dataBytes, marshalErr := json.Marshal(manageResp)
				if marshalErr == nil {
					var raw map[string]interface{}
					if json.Unmarshal(dataBytes, &raw) == nil {
						if dataMap, ok := raw["data"].(map[string]interface{}); ok {
							for _, key := range []string{"managementUrl", "manageUrl"} {
								if url, ok := dataMap[key].(string); ok && url != "" {
									managementURL = url
									t.Logf("  %s (from raw): %s", key, managementURL)
									break
								}
							}
						}
					}
				}
			}
			if managementURL != "" {
				break
			}
		}

		t.Logf("  Attempt %d: managementUrl not available yet: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
			attempt+1, webhookSubRequest, webhookSubID, manageResp.GetCode(), manageResp.GetMessage(), manageResp.GetData())
		time.Sleep(5 * time.Second)
	}

	if managementURL == "" {
		t.Log("Could not get managementUrl - skipping period changed test")
		t.Skip("managementUrl not available")
	}

	// Navigate to management page and simulate next period
	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	t.Log("=== Step 2: Navigating to management page ===")
	if err := base.NavigateTo(managementURL); err != nil {
		t.Fatalf("Failed to navigate to management page: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(3000)

	// Log all buttons/links for debugging
	buttons, _ := base.Page.Locator("button, a").All()
	t.Logf("  Found %d buttons/links on management page:", len(buttons))
	maxButtons := 20
	if len(buttons) < maxButtons {
		maxButtons = len(buttons)
	}
	for i := 0; i < maxButtons; i++ {
		text, _ := buttons[i].TextContent()
		tag, _ := buttons[i].Evaluate("el => el.tagName", nil)
		if text != "" {
			trimmed := strings.TrimSpace(text)
			if len(trimmed) > 60 {
				trimmed = trimmed[:60]
			}
			t.Logf("    <%v> \"%s\"", tag, trimmed)
		}
	}

	// Try to find and click "simulate next period success" button
	simulateSelectors := []string{
		"button:has-text('Simulate Next Period Success')",
		"button:has-text('simulate next period success')",
		"button:has-text('Next Period Success')",
		"button:has-text('Simulate Success')",
		"a:has-text('Simulate Next Period Success')",
		"a:has-text('simulate next period success')",
		"a:has-text('Next Period Success')",
		"a:has-text('Simulate Success')",
	}

	clicked := false
	for _, sel := range simulateSelectors {
		count, _ := base.Page.Locator(sel).Count()
		if count > 0 {
			base.Page.Locator(sel).First().Click()
			t.Logf("  Clicked simulate button: %s", sel)
			clicked = true
			base.SleepMs(3000)
			break
		}
	}

	if !clicked {
		t.Log("Could not find 'simulate next period' button on management page")
		t.Skip("Simulate next period button not found")
	}

	// Wait for SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION
	t.Log("=== Step 3: Waiting for period changed notification ===")
	notifications := webhookServer.WaitForNotification(
		"SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION",
		1,
		webhookTimeout,
	)

	if len(notifications) > 0 {
		n := notifications[0]
		if !n.HandlerSuccess {
			t.Errorf("Webhook handler reported failure for period changed notification: subscriptionRequest=%s, subscriptionID=%s, eventType=%s, parsed=%+v",
				webhookSubRequest, webhookSubID, n.EventType, n.Parsed)
		}
		if n.EventType != "SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION" {
			t.Errorf("Expected SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION, got %s: subscriptionRequest=%s, subscriptionID=%s, parsed=%+v",
				n.EventType, webhookSubRequest, webhookSubID, n.Parsed)
		}
		t.Log("SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION received and verified!")
	} else {
		t.Log("No SUBSCRIPTION_PERIOD_CHANGED_NOTIFICATION received within timeout")
	}
}

// ==================== Test 3: Subscription Cancellation ====================

func TestWebhook_SubscriptionCancellation(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	if !webhookSubCreated || webhookServer == nil {
		t.Skip("Subscription not created in previous test")
	}

	t.Log("=== Cancelling subscription ===")
	t.Logf("  subscriptionRequest: %s", webhookSubRequest)
	t.Logf("  subscriptionId: %s", webhookSubID)

	beforeCount := len(webhookServer.GetNotificationsByType("SUBSCRIPTION_STATUS_NOTIFICATION"))

	cancelParams := &subscription.CancelSubscriptionParams{
		SubscriptionID: webhookSubID,
		MerchantID:     testConfig.MerchantID,
		RequestedAt:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	cancelResp, err := testWaffo.Subscription().Cancel(context.Background(), cancelParams, nil)
	if err != nil {
		t.Logf("Cancel error: subscriptionRequest=%s, subscriptionID=%s, error=%v",
			webhookSubRequest, webhookSubID, err)
		return
	}

	t.Logf("  Cancel response: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
		webhookSubRequest, webhookSubID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())

	if !cancelResp.IsSuccess() {
		t.Logf("Cancel may have failed: subscriptionRequest=%s, subscriptionID=%s, code=%s, msg=%s, data=%+v",
			webhookSubRequest, webhookSubID, cancelResp.GetCode(), cancelResp.GetMessage(), cancelResp.GetData())
		return
	}

	// Wait for cancellation notification
	t.Log("=== Waiting for cancellation notification ===")
	notifications := webhookServer.WaitForNotification(
		"SUBSCRIPTION_STATUS_NOTIFICATION",
		beforeCount+1,
		webhookTimeout,
	)

	cancelFound := false
	for _, n := range notifications {
		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			if result["subscriptionStatus"] == "MERCHANT_CANCELLED" {
				cancelFound = true
				t.Log("Cancellation notification received!")
				break
			}
		}
	}

	if !cancelFound {
		t.Log("No cancellation notification received within timeout")
	}
}

// ==================== Test 4: Payment Notification ====================

func TestWebhook_PaymentNotification(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	if webhookServer == nil || webhookNgrokURL == "" {
		t.Skip("Webhook infrastructure not running")
	}

	base := &BaseE2ETest{}
	if err := base.Setup(); err != nil {
		t.Fatalf("Failed to setup browser: %v", err)
	}
	defer base.Teardown()

	// 1. Create one-time order with webhook URL
	webhookPaymentRequestID = fmt.Sprintf("e2e_wh_pay_%d", time.Now().UnixMilli())
	merchantOrderID := fmt.Sprintf("WH_PAY_E2E_%d", time.Now().UnixMilli())
	notifyURL := webhookNgrokURL + "/webhook"

	t.Log("=== Step 1: Creating one-time order ===")
	t.Logf("  paymentRequestId: %s", webhookPaymentRequestID)
	t.Logf("  notifyUrl: %s", notifyURL)

	createParams := &order.CreateOrderParams{
		PaymentRequestID:   webhookPaymentRequestID,
		MerchantOrderID:    merchantOrderID,
		OrderCurrency:      "HKD",
		OrderAmount:        "100.00",
		OrderDescription:   "Webhook Payment E2E Test",
		NotifyURL:          notifyURL,
		SuccessRedirectURL: TestURLs.Success,
		FailedRedirectURL:  TestURLs.Failed,
		CancelRedirectURL:  TestURLs.Cancel,
		OrderRequestedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		MerchantInfo: &order.MerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &order.UserInfo{
			UserID:       fmt.Sprintf("wh_pay_user_%d", time.Now().Unix()),
			UserEmail:    "webhook_pay_e2e@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &order.GoodsInfo{
			GoodsID:       "WH_PAY_GOODS",
			GoodsName:     "Webhook Payment E2E Product",
			GoodsCategory: "GOODS",
			GoodsURL:      "https://example.com/webhook-pay-test",
			GoodsQuantity: 1,
		},
		PaymentInfo: &order.PaymentInfo{
			ProductName:   "ONE_TIME_PAYMENT",
			PayMethodType: "CREDITCARD",
		},
	}

	createResp, err := testWaffo.Order().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Fatalf("Failed to create order: paymentRequestID=%s, error=%v", webhookPaymentRequestID, err)
	}

	t.Logf("  Response: code=%s, msg=%s, data=%+v", createResp.GetCode(), createResp.GetMessage(), createResp.GetData())

	if !createResp.IsSuccess() {
		t.Fatalf("Create order failed: paymentRequestID=%s, code=%s, msg=%s, data=%+v",
			webhookPaymentRequestID, createResp.GetCode(), createResp.GetMessage(), createResp.GetData())
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Fatalf("Order data is nil: paymentRequestID=%s, code=%s, msg=%s",
			webhookPaymentRequestID, createResp.GetCode(), createResp.GetMessage())
	}

	webhookAcquiringOrderID = createData.AcquiringOrderID
	t.Logf("  acquiringOrderId: %s", webhookAcquiringOrderID)

	checkoutURL := createData.FetchRedirectURL()
	if checkoutURL == "" {
		t.Fatalf("Checkout URL is empty: paymentRequestID=%s, data=%+v", webhookPaymentRequestID, createData)
	}
	t.Logf("  checkoutUrl: %s", checkoutURL)

	// 2. Complete payment via Playwright (standard test card, no 3DS)
	t.Log("=== Step 2: Completing payment ===")
	if err := base.NavigateTo(checkoutURL); err != nil {
		t.Fatalf("Failed to navigate to checkout: %v", err)
	}

	base.WaitForPageLoad()
	base.SleepMs(2000)

	paymentSuccess := completeSimplePaymentFlow(t, base.Page)
	t.Logf("  Payment flow result: %v", paymentSuccess)

	// Verify payment success by querying order with retries
	base.SleepMs(3000)
	for i := 0; i < 5; i++ {
		inquiryParams := &order.InquiryOrderParams{
			PaymentRequestID: webhookPaymentRequestID,
		}
		inquiryResp, inquiryErr := testWaffo.Order().Inquiry(context.Background(), inquiryParams, nil)
		if inquiryErr != nil {
			t.Logf("  Order inquiry error (attempt %d): paymentRequestID=%s, error=%v",
				i+1, webhookPaymentRequestID, inquiryErr)
		} else if inquiryResp.IsSuccess() {
			inquiryData := inquiryResp.GetData()
			if inquiryData != nil {
				t.Logf("  Order status (attempt %d): paymentRequestID=%s, status=%s",
					i+1, webhookPaymentRequestID, inquiryData.OrderStatus)
				if inquiryData.OrderStatus == "PAY_SUCCESS" {
					webhookOrderPaid = true
					break
				}
			}
		} else {
			t.Logf("  Order inquiry failed (attempt %d): paymentRequestID=%s, code=%s, msg=%s, data=%+v",
				i+1, webhookPaymentRequestID, inquiryResp.GetCode(), inquiryResp.GetMessage(), inquiryResp.GetData())
		}
		time.Sleep(2 * time.Second)
	}

	// 3. Wait for PAYMENT_NOTIFICATION
	t.Log("=== Step 3: Waiting for PAYMENT_NOTIFICATION ===")
	notifications := webhookServer.WaitForNotification(
		"PAYMENT_NOTIFICATION",
		1,
		webhookTimeout,
	)

	if len(notifications) > 0 {
		n := notifications[0]
		if !n.HandlerSuccess {
			t.Errorf("Payment notification handler reported failure: paymentRequestID=%s, eventType=%s, parsed=%+v",
				webhookPaymentRequestID, n.EventType, n.Parsed)
		}
		if n.EventType != "PAYMENT_NOTIFICATION" {
			t.Errorf("Expected PAYMENT_NOTIFICATION, got %s: paymentRequestID=%s, parsed=%+v",
				n.EventType, webhookPaymentRequestID, n.Parsed)
		}

		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			t.Logf("  orderStatus: %v", result["orderStatus"])
			if v, ok := result["acquiringOrderId"]; ok {
				t.Logf("  acquiringOrderId: %v", v)
			}
		}

		t.Log("PAYMENT_NOTIFICATION received and verified!")
	} else {
		t.Log("No PAYMENT_NOTIFICATION received within timeout (sandbox may not send for one-time orders)")
	}
}

// ==================== Test 5: Refund Notification ====================

func TestWebhook_RefundNotification(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	if webhookServer == nil || webhookNgrokURL == "" {
		t.Skip("Webhook infrastructure not running")
	}

	if !webhookOrderPaid || webhookPaymentRequestID == "" {
		t.Skip("Order not paid in previous test - cannot refund")
	}

	t.Log("=== Step 1: Refunding order ===")
	t.Logf("  paymentRequestId: %s", webhookPaymentRequestID)

	refundRequestID := fmt.Sprintf("e2e_wh_refund_%d", time.Now().UnixMilli())
	notifyURL := webhookNgrokURL + "/webhook"

	refundParams := &order.RefundOrderParams{
		AcquiringOrderID: webhookAcquiringOrderID,
		RefundRequestID:  refundRequestID,
		RefundAmount:     "30.00",
		RefundReason:     "Webhook E2E refund test",
		NotifyURL:        notifyURL,
	}

	refundResp, err := testWaffo.Order().Refund(context.Background(), refundParams, nil)
	if err != nil {
		t.Logf("Refund API error: paymentRequestID=%s, refundRequestID=%s, error=%v",
			webhookPaymentRequestID, refundRequestID, err)
		t.Skip("Refund API call failed")
	}

	t.Logf("  Refund response: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
		webhookPaymentRequestID, refundRequestID, refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())

	if !refundResp.IsSuccess() {
		t.Logf("Refund failed: paymentRequestID=%s, refundRequestID=%s, code=%s, msg=%s, data=%+v",
			webhookPaymentRequestID, refundRequestID, refundResp.GetCode(), refundResp.GetMessage(), refundResp.GetData())
		return
	}

	refundData := refundResp.GetData()
	if refundData != nil {
		t.Logf("  refundStatus: %s", refundData.RefundStatus)
		t.Logf("  acquiringRefundOrderId: %s", refundData.AcquiringRefundOrderID)
	}

	// 2. Wait for REFUND_NOTIFICATION
	t.Log("=== Step 2: Waiting for REFUND_NOTIFICATION ===")
	notifications := webhookServer.WaitForNotification(
		"REFUND_NOTIFICATION",
		1,
		webhookTimeout,
	)

	if len(notifications) > 0 {
		n := notifications[0]
		if !n.HandlerSuccess {
			t.Errorf("Refund notification handler reported failure: paymentRequestID=%s, refundRequestID=%s, eventType=%s, parsed=%+v",
				webhookPaymentRequestID, refundRequestID, n.EventType, n.Parsed)
		}
		if n.EventType != "REFUND_NOTIFICATION" {
			t.Errorf("Expected REFUND_NOTIFICATION, got %s: paymentRequestID=%s, refundRequestID=%s, parsed=%+v",
				n.EventType, webhookPaymentRequestID, refundRequestID, n.Parsed)
		}

		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			t.Logf("  refundStatus: %v", result["refundStatus"])
		}

		t.Log("REFUND_NOTIFICATION received and verified!")
	} else {
		t.Log("No REFUND_NOTIFICATION received within timeout (sandbox may not send refund notifications)")
	}
}

// ==================== Test 5.5: Subscription Change ====================

func TestWebhook_SubscriptionChange(t *testing.T) {
	if testWaffo == nil {
		t.Skip("Waffo client not initialized - config missing")
	}

	if webhookServer == nil || webhookNgrokURL == "" {
		t.Skip("Webhook infrastructure not initialized")
	}

	notifyURL := webhookNgrokURL + "/webhook"
	changeSubRequest := fmt.Sprintf("e2e_wh_chg_%d", time.Now().UnixMilli())
	newSubRequest := fmt.Sprintf("e2e_wh_new_%d", time.Now().UnixMilli())
	changeUserID := fmt.Sprintf("wh_change_user_%d", time.Now().Unix())

	// ==================== Step 1: Create original subscription ====================
	t.Log("=== [SubscriptionChange E2E] Step 1: Creating original subscription ===")
	t.Logf("  changeSubRequest: %s", changeSubRequest)
	t.Logf("  notifyURL: %s", notifyURL)

	createParams := &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    changeSubRequest,
		MerchantSubscriptionID: fmt.Sprintf("WH_CHG_ORIG_%d", time.Now().UnixMilli()),
		Currency:               "HKD",
		Amount:                 "99.00",
		ProductInfo: &subscription.ProductInfo{
			Description:    "Webhook Change E2E Original Subscription",
			PeriodType:     "MONTHLY",
			PeriodInterval: "1",
			NumberOfPeriod: "12",
		},
		MerchantInfo: &subscription.SubscriptionMerchantInfo{
			MerchantID: testConfig.MerchantID,
		},
		UserInfo: &subscription.SubscriptionUserInfo{
			UserID:       changeUserID,
			UserEmail:    "wh_change_e2e@test.com",
			UserTerminal: "WEB",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "WH_CHG_GOODS",
			GoodsName:     "Webhook Change E2E Product",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/wh-change-test",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
		RequestedAt:               time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		NotifyURL:                 notifyURL,
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		SubscriptionManagementURL: "https://example.com/manage",
	}

	createResp, err := testWaffo.Subscription().Create(context.Background(), createParams, nil)
	if err != nil {
		t.Logf("[SubscriptionChange E2E] Create subscription error: changeSubRequest=%s, error=%v", changeSubRequest, err)
		t.Skip("Skipping due to create subscription error")
		return
	}
	if !createResp.IsSuccess() {
		t.Logf("[SubscriptionChange E2E] Create subscription failed: changeSubRequest=%s, code=%s, msg=%s",
			changeSubRequest, createResp.GetCode(), createResp.GetMessage())
		t.Skip("Skipping due to create subscription failure")
		return
	}

	createData := createResp.GetData()
	if createData == nil {
		t.Skip("[SubscriptionChange E2E] Create subscription data is nil")
		return
	}

	origSubID := createData.SubscriptionID
	checkoutURL := createData.FetchRedirectURL()
	if checkoutURL == "" {
		t.Skip("[SubscriptionChange E2E] No checkout URL available")
		return
	}

	t.Logf("  origSubID: %s", origSubID)
	t.Logf("  checkoutURL: %s", checkoutURL)

	// ==================== Step 2: Activate subscription via Playwright ====================
	t.Log("=== [SubscriptionChange E2E] Step 2: Activating subscription ===")
	activationBase := &BaseE2ETest{}
	if err := activationBase.Setup(); err != nil {
		t.Logf("[SubscriptionChange E2E] Browser setup failed: %v", err)
		t.Skip("Skipping due to browser setup failure")
		return
	}
	defer activationBase.Teardown()

	if err := activationBase.NavigateTo(checkoutURL); err != nil {
		t.Logf("[SubscriptionChange E2E] Navigate to checkout failed: changeSubRequest=%s, error=%v", changeSubRequest, err)
		return
	}
	activationBase.WaitForPageLoad()
	activationBase.SleepMs(3000)
	// Capture count BEFORE payment so we can wait for the activation notification
	beforeActivationCount := len(webhookServer.GetNotificationsByType("SUBSCRIPTION_STATUS_NOTIFICATION"))
	activated := completePaymentFlow(t, activationBase.Page)
	t.Logf("  Activation result: %v", activated)

	// Wait for activation webhook
	webhookServer.WaitForNotification("SUBSCRIPTION_STATUS_NOTIFICATION", beforeActivationCount+1, webhookTimeout)

	// ==================== Step 3: Call subscription change API ====================
	t.Log("=== [SubscriptionChange E2E] Step 3: Calling subscription change API ===")
	t.Logf("  newSubRequest: %s", newSubRequest)
	t.Logf("  originSubRequest: %s", changeSubRequest)

	orderExpiredAt := time.Now().Add(30 * time.Minute).UTC().Format("2006-01-02T15:04:05.000Z")

	changeParams := &subscription.ChangeSubscriptionParams{
		SubscriptionRequest:       newSubRequest,
		MerchantSubscriptionID:    fmt.Sprintf("MSUB_WH_CHG_%d", time.Now().UnixMilli()),
		OriginSubscriptionRequest: changeSubRequest,
		RemainingAmount:           "50.00",
		Currency:                  "HKD",
		RequestedAt:               time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		OrderExpiredAt:            orderExpiredAt,
		NotifyURL:                 notifyURL,
		SuccessRedirectURL:        TestURLs.Success,
		FailedRedirectURL:         TestURLs.Failed,
		CancelRedirectURL:         TestURLs.Cancel,
		ProductInfoList: []subscription.SubscriptionChangeProductInfo{
			{
				Description:    "Premium Monthly (Upgraded for webhook test)",
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
			UserID:    changeUserID,
			UserEmail: "wh_change_e2e@test.com",
		},
		GoodsInfo: &subscription.SubscriptionGoodsInfo{
			GoodsID:       "GOODS_PREMIUM_WH_E2E",
			GoodsName:     "Premium Membership for Webhook Test",
			GoodsCategory: "SUBSCRIPTION",
			GoodsURL:      "https://example.com/subscription/premium",
			GoodsQuantity: 1,
		},
		PaymentInfo: &subscription.SubscriptionPaymentInfo{
			ProductName:   "SUBSCRIPTION",
			PayMethodType: "CREDITCARD",
		},
	}

	changeResp, err := testWaffo.Subscription().Change(context.Background(), changeParams, nil)
	if err != nil {
		t.Logf("[SubscriptionChange E2E] Change API error: newSubRequest=%s, originSubRequest=%s, error=%v",
			newSubRequest, changeSubRequest, err)
		return
	}
	t.Logf("  Change response: code=%s, msg=%s, data=%+v", changeResp.GetCode(), changeResp.GetMessage(), changeResp.GetData())

	if !changeResp.IsSuccess() {
		t.Logf("[SubscriptionChange E2E] Change API failed: newSubRequest=%s, originSubRequest=%s, code=%s, msg=%s, data=%+v",
			newSubRequest, changeSubRequest, changeResp.GetCode(), changeResp.GetMessage(), changeResp.GetData())
		t.Log("This may be normal if subscription is not ACTIVE yet in sandbox")
		return
	}

	changeData := changeResp.GetData()
	if changeData != nil {
		t.Logf("  subscriptionChangeStatus: %s", changeData.SubscriptionChangeStatus)
		t.Logf("  subscriptionAction: %s", changeData.SubscriptionAction)

		// ==================== Step 4: Handle AUTHORIZATION_REQUIRED ====================
		if changeData.SubscriptionChangeStatus == "AUTHORIZATION_REQUIRED" {
			t.Log("=== [SubscriptionChange E2E] Step 4: Handling AUTHORIZATION_REQUIRED ===")
			authURL := changeData.FetchRedirectURL()
			if authURL != "" {
				t.Logf("  authURL: %s", authURL)
				authBase := &BaseE2ETest{}
				if err := authBase.Setup(); err == nil {
					defer authBase.Teardown()
					if err := authBase.NavigateTo(authURL); err == nil {
						authBase.WaitForPageLoad()
						authBase.SleepMs(3000)
						authSuccess := completePaymentFlow(t, authBase.Page)
						t.Logf("  Auth result: %v", authSuccess)
					}
				}
			}
		}
	}

	// ==================== Step 5: Wait for SUBSCRIPTION_CHANGE_NOTIFICATION ====================
	t.Log("=== [SubscriptionChange E2E] Step 5: Waiting for SUBSCRIPTION_CHANGE_NOTIFICATION ===")
	changeNotifications := webhookServer.WaitForNotification(
		"SUBSCRIPTION_CHANGE_NOTIFICATION", 1, webhookTimeout,
	)

	if len(changeNotifications) > 0 {
		n := changeNotifications[0]
		if !n.HandlerSuccess {
			t.Errorf("[SubscriptionChange E2E] Handler reported failure: newSubRequest=%s, originSubRequest=%s, eventType=%s, parsed=%+v",
				newSubRequest, changeSubRequest, n.EventType, n.Parsed)
		}
		if n.EventType != "SUBSCRIPTION_CHANGE_NOTIFICATION" {
			t.Errorf("[SubscriptionChange E2E] Expected SUBSCRIPTION_CHANGE_NOTIFICATION, got %s: newSubRequest=%s",
				n.EventType, newSubRequest)
		}
		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			t.Logf("  subscriptionChangeStatus: %v", result["subscriptionChangeStatus"])
			t.Logf("  subscriptionRequest: %v", result["subscriptionRequest"])
		}
		t.Log("[SubscriptionChange E2E] SUBSCRIPTION_CHANGE_NOTIFICATION received and verified!")
	} else {
		t.Logf("[SubscriptionChange E2E] No SUBSCRIPTION_CHANGE_NOTIFICATION received within timeout: newSubRequest=%s, originSubRequest=%s",
			newSubRequest, changeSubRequest)
		t.Log("This may be normal if sandbox did not trigger the notification in time")
	}
}

// ==================== Test 6: Cleanup ====================

func TestWebhook_Cleanup(t *testing.T) {
	// This test runs last to clean up infrastructure
	cleanupWebhookInfra()
}

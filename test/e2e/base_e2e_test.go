//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// BaseE2ETest provides common functionality for E2E tests
type BaseE2ETest struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	Page    playwright.Page
}

// TestCard contains sandbox test card information
var TestCard = struct {
	Number      string
	ExpiryMonth string
	ExpiryYear  string
	CVV         string
	Holder      string
}{
	Number:      "4111 1111 1111 1111",
	ExpiryMonth: "09",
	ExpiryYear:  "2028",
	CVV:         "123",
	Holder:      "Tom Clause",
}

// CheckoutCard contains the browser-facing card fields used by checkout helpers.
type CheckoutCard struct {
	Number      string
	ExpiryMonth string
	ExpiryYear  string
	CVV         string
	Holder      string
}

// TestURLs contains redirect URLs for testing
var TestURLs = struct {
	Success string
	Failed  string
	Cancel  string
}{
	Success: "https://httpbin.org/get?status=success",
	Failed:  "https://httpbin.org/get?status=failed",
	Cancel:  "https://httpbin.org/get?status=cancel",
}

// IsHeadless returns whether to run browser in headless mode
func IsHeadless() bool {
	val := os.Getenv("E2E_HEADLESS")
	if val == "" {
		return true // Default to headless
	}
	headless, _ := strconv.ParseBool(val)
	return headless
}

// GetDefaultTimeout returns the default timeout in milliseconds
func GetDefaultTimeout() float64 {
	val := os.Getenv("E2E_TIMEOUT")
	if val == "" {
		return 30000 // Default 30 seconds
	}
	timeout, _ := strconv.ParseFloat(val, 64)
	return timeout
}

// IsScreenshotEnabled returns whether to take screenshots
func IsScreenshotEnabled() bool {
	val := os.Getenv("E2E_SCREENSHOT")
	if val == "" {
		return false
	}
	enabled, _ := strconv.ParseBool(val)
	return enabled
}

// GetScreenshotPath returns the path for saving screenshots
func GetScreenshotPath() string {
	path := os.Getenv("E2E_SCREENSHOT_PATH")
	if path == "" {
		return "test-screenshots"
	}
	return path
}

// GetBrowserType returns the browser type to use
func GetBrowserType() string {
	browser := os.Getenv("E2E_BROWSER")
	if browser == "" {
		return "firefox" // Default to firefox
	}
	return browser
}

// Setup initializes the browser and page
func (b *BaseE2ETest) Setup() error {
	var err error

	if os.Getenv("E2E_INSTALL_BROWSERS") == "true" {
		if err = playwright.Install(); err != nil {
			fmt.Printf("Playwright install note: %v\n", err)
		}
	}

	b.pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %v", err)
	}

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(IsHeadless()),
	}

	if !IsHeadless() {
		launchOptions.SlowMo = playwright.Float(100) // Slow down for visibility
	}

	b.browser, err = b.launchBrowserWithFallback(launchOptions)
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}

	// Create context with viewport
	b.context, err = b.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1280,
			Height: 720,
		},
		Locale: playwright.String("en-US"),
	})
	if err != nil {
		return fmt.Errorf("could not create context: %v", err)
	}

	// Create page
	b.Page, err = b.context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %v", err)
	}

	b.Page.SetDefaultTimeout(GetDefaultTimeout())

	return nil
}

func (b *BaseE2ETest) launchBrowserWithFallback(launchOptions playwright.BrowserTypeLaunchOptions) (playwright.Browser, error) {
	requested := GetBrowserType()
	candidates := []string{requested}
	if os.Getenv("E2E_BROWSER") == "" {
		for _, candidate := range []string{"chromium", "webkit", "firefox"} {
			if !containsString(candidates, candidate) {
				candidates = append(candidates, candidate)
			}
		}
	}

	var lastErr error
	for _, candidate := range candidates {
		browser, err := b.launchBrowser(candidate, launchOptions)
		if err == nil {
			if candidate != requested {
				fmt.Printf("Playwright browser fallback: %s -> %s\n", requested, candidate)
			}
			return browser, nil
		}
		lastErr = err
		fmt.Printf("Playwright launch failed for %s: %v\n", candidate, err)
	}

	return nil, lastErr
}

func (b *BaseE2ETest) launchBrowser(browserType string, launchOptions playwright.BrowserTypeLaunchOptions) (playwright.Browser, error) {
	switch browserType {
	case "firefox":
		return b.pw.Firefox.Launch(launchOptions)
	case "webkit":
		return b.pw.WebKit.Launch(launchOptions)
	default:
		return b.pw.Chromium.Launch(launchOptions)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Teardown closes the browser and cleans up resources
func (b *BaseE2ETest) Teardown() {
	if b.context != nil {
		b.context.Close()
	}
	if b.browser != nil {
		b.browser.Close()
	}
	if b.pw != nil {
		b.pw.Stop()
	}
}

// NavigateTo navigates to the specified URL
func (b *BaseE2ETest) NavigateTo(url string) error {
	_, err := b.Page.Goto(url)
	return err
}

// WaitForPageLoad waits for the page to finish loading
func (b *BaseE2ETest) WaitForPageLoad() error {
	return b.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
}

// Click clicks on an element matching the selector
func (b *BaseE2ETest) Click(selector string) error {
	return b.Page.Click(selector)
}

// ClickButton clicks a button with the specified text
func (b *BaseE2ETest) ClickButton(buttonText string) error {
	selector := fmt.Sprintf("button:has-text('%s')", buttonText)
	return b.Page.Click(selector)
}

// Fill fills an input field with the specified value
func (b *BaseE2ETest) Fill(selector, value string) error {
	return b.Page.Fill(selector, value)
}

// GetText returns the text content of an element
func (b *BaseE2ETest) GetText(selector string) (string, error) {
	return b.Page.TextContent(selector)
}

// IsVisible checks if an element is visible
func (b *BaseE2ETest) IsVisible(selector string) (bool, error) {
	return b.Page.IsVisible(selector)
}

// WaitForSelector waits for an element to appear
func (b *BaseE2ETest) WaitForSelector(selector string) error {
	_, err := b.Page.WaitForSelector(selector)
	return err
}

// WaitForURL waits for the URL to match the specified pattern
func (b *BaseE2ETest) WaitForURL(pattern string) error {
	return b.Page.WaitForURL(pattern)
}

// WaitForURLContains waits for the URL to contain the specified string
func (b *BaseE2ETest) WaitForURLContains(urlPart string) error {
	return b.Page.WaitForURL("**" + urlPart + "**")
}

// TakeScreenshot saves a screenshot with the given name
func (b *BaseE2ETest) TakeScreenshot(name string) error {
	if !IsScreenshotEnabled() {
		return nil
	}

	// Ensure directory exists
	dir := GetScreenshotPath()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, name+".png")
	_, err := b.Page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	return err
}

// PrintCurrentURL prints the current page URL
func (b *BaseE2ETest) PrintCurrentURL() {
	fmt.Printf("Current URL: %s\n", b.Page.URL())
}

// Sleep pauses execution for the specified duration
func (b *BaseE2ETest) Sleep(d time.Duration) {
	time.Sleep(d)
}

// SleepMs pauses execution for the specified milliseconds
func (b *BaseE2ETest) SleepMs(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// SetTimeout sets the default timeout for the page
func (b *BaseE2ETest) SetTimeout(timeout float64) {
	b.Page.SetDefaultTimeout(timeout)
}

// DefaultCheckoutCard returns the standard sandbox test card.
func DefaultCheckoutCard() CheckoutCard {
	return CheckoutCard{
		Number:      TestCard.Number,
		ExpiryMonth: TestCard.ExpiryMonth,
		ExpiryYear:  TestCard.ExpiryYear,
		CVV:         TestCard.CVV,
		Holder:      TestCard.Holder,
	}
}

// SubscriptionCheckoutCard matches the card used by stable subscription E2E flows.
func SubscriptionCheckoutCard() CheckoutCard {
	return CheckoutCard{
		Number:      "4000000000001000",
		ExpiryMonth: "12",
		ExpiryYear:  "2028",
		CVV:         TestCard.CVV,
		Holder:      "Tom",
	}
}

// CompleteCheckout navigates to a checkout URL and completes the browser payment flow.
func (b *BaseE2ETest) CompleteCheckout(t *testing.T, checkoutURL, screenshotPrefix string, card CheckoutCard) bool {
	t.Helper()

	if err := b.NavigateTo(checkoutURL); err != nil {
		t.Logf("Failed to navigate to checkout: url=%s, error=%v", checkoutURL, err)
		return false
	}
	if err := b.WaitForPageLoad(); err != nil {
		t.Logf("Checkout page load wait failed: url=%s, error=%v", checkoutURL, err)
	}
	b.SleepMs(2000)
	b.TakeScreenshot(screenshotPrefix + "_loaded")

	ok := b.CompleteCurrentCheckout(t, card)
	b.TakeScreenshot(screenshotPrefix + "_completed")
	return ok
}

// CompleteCurrentCheckout completes the currently loaded checkout or authorization page.
func (b *BaseE2ETest) CompleteCurrentCheckout(t *testing.T, card CheckoutCard) bool {
	t.Helper()

	t.Logf("  Checkout current URL: %s", b.Page.URL())

	if !b.fillFullCardDetails(t, card) {
		t.Log("  Full card form not completed; trying saved-card/CVV-only authorization")
		b.printActionCandidates(t)
		b.selectSavedPaymentMethod(t)
		if !b.fillCVVOnly(t, card.CVV) {
			t.Log("  CVV-only input not visible; continuing with available submit/confirm controls")
		}
	}

	b.checkAllCheckboxes(t)

	if !b.clickFirstVisible(t, []string{
		"button[type='submit']",
		"input[type='submit']",
		"button:has-text('Pay')",
		"button:has-text('Submit')",
		"button:has-text('Confirm')",
		"button:has-text('Continue')",
		"button:has-text('支付')",
		"button:has-text('確認')",
		"button:has-text('确认')",
		"button",
	}, "submit/confirm") {
		return false
	}

	b.SleepMs(3000)
	b.acceptTermsIfPresent(t)
	b.handle3DSIfPresent(t)
	return b.waitForCheckoutResult(t)
}

func (b *BaseE2ETest) fillFullCardDetails(t *testing.T, card CheckoutCard) bool {
	t.Helper()

	cardNumberFilled := b.fillFirstVisible(t, []string{
		"input[name='payMethodProperties.card.pan']",
		"#payMethodProperties\\.card\\.pan",
		"input[name='cardNumber']",
		"input[placeholder*='card']",
		"input[placeholder*='Card']",
		"input[autocomplete='cc-number']",
		"#cardNumber",
		"[data-testid='card-number']",
	}, card.Number, "card number")
	if !cardNumberFilled {
		b.printAvailableInputs(t)
		return false
	}

	b.SleepMs(500)
	expiry := fmt.Sprintf("%s/%s", card.ExpiryMonth, card.ExpiryYear[2:])
	b.fillFirstVisible(t, []string{
		"input[name='payMethodProperties.card.expiry']",
		"#payMethodProperties\\.card\\.expiry",
		"input[name='expiry']",
		"input[name='expiryDate']",
		"input[placeholder*='MM']",
		"input[autocomplete='cc-exp']",
		"#expiry",
	}, expiry, "expiry")

	b.SleepMs(500)
	b.fillFirstVisible(t, []string{
		"input[name='payMethodProperties.card.cvv']",
		"#payMethodProperties\\.card\\.cvv",
		"input[name='cvv']",
		"input[name='cvc']",
		"input[name='securityCode']",
		"input[placeholder*='CVV']",
		"input[placeholder*='CVC']",
		"input[autocomplete='cc-csc']",
		"#cvv",
	}, card.CVV, "CVV")

	b.SleepMs(500)
	b.fillFirstVisible(t, []string{
		"input[name='payMethodProperties.card.name']",
		"#payMethodProperties\\.card\\.name",
		"input[name='cardholderName']",
		"input[name='name']",
		"input[placeholder*='name']",
		"input[placeholder*='Name']",
		"input[autocomplete='cc-name']",
		"#cardholderName",
	}, card.Holder, "cardholder name")

	b.SleepMs(500)
	return true
}

func (b *BaseE2ETest) fillCVVOnly(t *testing.T, cvv string) bool {
	t.Helper()

	return b.fillFirstVisible(t, []string{
		"input[name='payMethodProperties.card.cvv']",
		"#payMethodProperties\\.card\\.cvv",
		"input[name='cvv']",
		"input[name='cvc']",
		"input[name='securityCode']",
		"input[placeholder*='CVV']",
		"input[placeholder*='CVC']",
		"input[autocomplete='cc-csc']",
		"#cvv",
	}, cvv, "saved-card CVV")
}

func (b *BaseE2ETest) fillFirstVisible(t *testing.T, selectors []string, value, fieldName string) bool {
	t.Helper()

	for _, selector := range selectors {
		locator := b.Page.Locator(selector)
		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		for i := 0; i < count; i++ {
			item := locator.Nth(i)
			visible, err := item.IsVisible()
			if err != nil || !visible {
				continue
			}
			if err := item.Fill(value); err != nil {
				t.Logf("  Failed to fill %s using %s: %v", fieldName, selector, err)
				continue
			}
			t.Logf("  %s filled using: %s", fieldName, selector)
			return true
		}
	}

	return false
}

func (b *BaseE2ETest) clickFirstVisible(t *testing.T, selectors []string, label string) bool {
	t.Helper()

	for _, selector := range selectors {
		locator := b.Page.Locator(selector)
		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		for i := 0; i < count; i++ {
			item := locator.Nth(i)
			visible, err := item.IsVisible()
			if err != nil || !visible {
				continue
			}
			text, _ := item.TextContent()
			if err := item.Click(); err != nil {
				t.Logf("  Failed to click %s using %s: %v", label, selector, err)
				continue
			}
			t.Logf("  Clicked %s using: %s text=%q", label, selector, strings.TrimSpace(text))
			return true
		}
	}

	t.Logf("  Could not find visible %s control", label)
	return false
}

func (b *BaseE2ETest) selectSavedPaymentMethod(t *testing.T) bool {
	t.Helper()

	savedCardSelectors := []string{
		"[data-dd-action-name='select_saved_card']",
		"button:has-text('ending')",
		"button:has-text('expires')",
		"button:has-text('到期')",
		"button:has-text('****')",
	}

	for _, selector := range savedCardSelectors {
		locator := b.Page.Locator(selector)
		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}
		for i := 0; i < count; i++ {
			item := locator.Nth(i)
			visible, err := item.IsVisible()
			if err != nil || !visible {
				continue
			}
			text, _ := item.TextContent()
			if err := item.Click(); err != nil {
				t.Logf("  Failed to select saved payment method using %s: %v", selector, err)
				continue
			}
			t.Logf("  Selected saved payment method: %q", strings.TrimSpace(text))
			b.SleepMs(1000)
			return true
		}
	}

	buttons, _ := b.Page.Locator("button, a").All()
	for _, button := range buttons {
		visible, err := button.IsVisible()
		if err != nil || !visible {
			continue
		}
		text, _ := button.TextContent()
		trimmed := strings.TrimSpace(text)
		normalized := strings.ToLower(trimmed)
		if strings.Contains(normalized, "ending") ||
			strings.Contains(normalized, "expires") ||
			strings.Contains(trimmed, "到期") ||
			strings.Contains(trimmed, "****") {
			if err := button.Click(); err != nil {
				t.Logf("  Failed to click saved payment candidate %q: %v", trimmed, err)
				continue
			}
			t.Logf("  Selected saved payment method: %q", trimmed)
			b.SleepMs(1000)
			return true
		}
	}

	return false
}

func (b *BaseE2ETest) checkAllCheckboxes(t *testing.T) {
	t.Helper()

	checkboxes, err := b.Page.Locator("input[type='checkbox']").All()
	if err != nil {
		t.Logf("  Failed to list checkboxes: %v", err)
		return
	}
	t.Logf("  Found %d checkbox(es)", len(checkboxes))
	for _, checkbox := range checkboxes {
		checked, err := checkbox.IsChecked()
		if err != nil || checked {
			continue
		}
		if err := checkbox.Check(); err != nil {
			t.Logf("  Failed to check checkbox: %v", err)
		}
	}
	b.SleepMs(500)
}

func (b *BaseE2ETest) acceptTermsIfPresent(t *testing.T) {
	t.Helper()

	b.clickFirstVisible(t, []string{
		"button:has-text('接受並繼續')",
		"button:has-text('接受并继续')",
		"button:has-text('Accept')",
		"button:has-text('Agree')",
		"button:has-text('Continue')",
		"button:has-text('確認')",
		"button:has-text('确认')",
	}, "terms accept")
}

func (b *BaseE2ETest) handle3DSIfPresent(t *testing.T) {
	t.Helper()

	b.SleepMs(3000)
	url := b.Page.URL()
	t.Logf("  URL after submit: %s", url)
	if strings.Contains(strings.ToLower(url), "dochallenge") || strings.Contains(strings.ToLower(url), "3ds") {
		b.fill3DSCode(t)
		return
	}

	iframeCount, _ := b.Page.Locator("iframe").Count()
	t.Logf("  Found %d iframe(s)", iframeCount)
	for i := 0; i < iframeCount; i++ {
		frame := b.Page.FrameLocator("iframe").Nth(i)
		for _, selector := range []string{
			"input[name='challengeDataEntry']",
			"input[name='otp']",
			"input[name='code']",
			"input[type='password']",
			"input[type='tel']",
		} {
			inputCount, _ := frame.Locator(selector).Count()
			if inputCount == 0 {
				continue
			}
			t.Logf("  3DS input found in iframe %d: %s", i, selector)
			frame.Locator(selector).First().Fill("1234")
			b.SleepMs(500)
			for _, submitSelector := range []string{
				"button[type='submit']",
				"input[type='submit']",
				"button:has-text('Submit')",
				"button:has-text('Verify')",
				"button:has-text('Confirm')",
				"button:has-text('確認')",
				"button:has-text('确认')",
			} {
				submitCount, _ := frame.Locator(submitSelector).Count()
				if submitCount == 0 {
					continue
				}
				frame.Locator(submitSelector).First().Click()
				t.Logf("  3DS submitted in iframe using: %s", submitSelector)
				b.SleepMs(5000)
				return
			}
		}
	}
}

func (b *BaseE2ETest) fill3DSCode(t *testing.T) {
	t.Helper()

	if !b.fillFirstVisible(t, []string{
		"input[name='challengeDataEntry']",
		"input[name='otp']",
		"input[name='code']",
		"input[type='password']",
		"input[type='tel']",
		"input[type='text']",
		"input",
	}, "1234", "3DS code") {
		return
	}

	b.SleepMs(500)
	b.clickFirstVisible(t, []string{
		"button[type='submit']",
		"input[type='submit']",
		"button:has-text('Submit')",
		"button:has-text('Verify')",
		"button:has-text('Confirm')",
		"button:has-text('確認')",
		"button:has-text('确认')",
		"button",
	}, "3DS submit")
	b.SleepMs(5000)
}

func (b *BaseE2ETest) waitForCheckoutResult(t *testing.T) bool {
	t.Helper()

	for i := 0; i < 15; i++ {
		b.SleepMs(2000)
		url := b.Page.URL()
		t.Logf("  Checkout result check %d/15: %s", i+1, url)

		normalizedURL := strings.ToLower(url)
		if strings.Contains(normalizedURL, "dochallenge") || strings.Contains(normalizedURL, "3ds") {
			t.Log("  Late 3DS challenge detected")
			b.fill3DSCode(t)
			continue
		}
		if strings.Contains(normalizedURL, "status=success") {
			t.Log("  Checkout success redirect detected")
			return true
		}
		if strings.Contains(normalizedURL, "status=failed") {
			t.Log("  Checkout failure redirect detected")
			return false
		}

		content, err := b.Page.Content()
		if err != nil {
			continue
		}
		if strings.Contains(content, "PAY_SUCCESS") ||
			strings.Contains(content, "支付成功") ||
			strings.Contains(content, "訂閱成功") ||
			strings.Contains(content, "success_page") {
			t.Log("  Checkout success indicator found in page content")
			return true
		}
	}

	t.Log("  Checkout result unknown after timeout")
	return false
}

func (b *BaseE2ETest) printActionCandidates(t *testing.T) {
	t.Helper()

	candidates, err := b.Page.Locator("button, a").All()
	if err != nil {
		t.Logf("  Failed to list action candidates: %v", err)
		return
	}
	t.Logf("  Found %d button/link candidate(s)", len(candidates))
	for i, candidate := range candidates {
		if i >= 20 {
			break
		}
		text, _ := candidate.TextContent()
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			t.Logf("    action candidate: %q", trimmed)
		}
	}
}

func (b *BaseE2ETest) printAvailableInputs(t *testing.T) {
	t.Helper()

	inputs, err := b.Page.Locator("input").All()
	if err != nil {
		t.Logf("  Failed to list inputs: %v", err)
		return
	}
	t.Logf("  Available input count: %d", len(inputs))
	for i, input := range inputs {
		if i >= 30 {
			break
		}
		inputType, _ := input.GetAttribute("type")
		name, _ := input.GetAttribute("name")
		placeholder, _ := input.GetAttribute("placeholder")
		t.Logf("    input: type=%s name=%s placeholder=%s", inputType, name, placeholder)
	}
}

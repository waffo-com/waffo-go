//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	// Install browsers if needed (first run only)
	if err = playwright.Install(); err != nil {
		// Ignore install errors if browsers are already installed
		fmt.Printf("Playwright install note: %v\n", err)
	}

	b.pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %v", err)
	}

	// Launch browser based on type
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(IsHeadless()),
	}

	if !IsHeadless() {
		launchOptions.SlowMo = playwright.Float(100) // Slow down for visibility
	}

	switch GetBrowserType() {
	case "firefox":
		b.browser, err = b.pw.Firefox.Launch(launchOptions)
	case "webkit":
		b.browser, err = b.pw.WebKit.Launch(launchOptions)
	default:
		b.browser, err = b.pw.Chromium.Launch(launchOptions)
	}

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

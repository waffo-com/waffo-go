//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/waffo-com/waffo-go/core"
)

// ReceivedNotification represents a webhook notification received by the test server.
type ReceivedNotification struct {
	EventType      string
	Body           string
	Parsed         map[string]interface{}
	Timestamp      time.Time
	HandlerSuccess bool
	ResponseBody   string
}

// WebhookTestServer is a local HTTP server for receiving webhook notifications.
type WebhookTestServer struct {
	server         *http.Server
	webhookHandler *core.WebhookHandler
	port           int
	notifications  []ReceivedNotification
	mu             sync.Mutex
}

// NewWebhookTestServer creates a new webhook test server.
func NewWebhookTestServer(handler *core.WebhookHandler, port int) *WebhookTestServer {
	return &WebhookTestServer{
		webhookHandler: handler,
		port:           port,
		notifications:  make([]ReceivedNotification, 0),
	}
}

// Start starts the webhook server.
func (s *WebhookTestServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-errCh:
		return fmt.Errorf("webhook server failed to start: %w", err)
	default:
		fmt.Printf("[WebhookServer] Listening on port %d\n", s.port)
		return nil
	}
}

// Stop stops the webhook server.
func (s *WebhookTestServer) Stop() {
	if s.server != nil {
		s.server.Close()
		fmt.Println("[WebhookServer] Stopped")
	}
}

// GetNotifications returns a copy of all received notifications.
func (s *WebhookTestServer) GetNotifications() []ReceivedNotification {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ReceivedNotification, len(s.notifications))
	copy(result, s.notifications)
	return result
}

// GetNotificationsByType returns notifications filtered by event type.
func (s *WebhookTestServer) GetNotificationsByType(eventType string) []ReceivedNotification {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []ReceivedNotification
	for _, n := range s.notifications {
		if n.EventType == eventType {
			result = append(result, n)
		}
	}
	return result
}

// WaitForNotification waits for a specific number of notifications of a given event type.
func (s *WebhookTestServer) WaitForNotification(eventType string, count int, timeout time.Duration) []ReceivedNotification {
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		matching := s.GetNotificationsByType(eventType)
		if len(matching) >= count {
			return matching
		}

		elapsed := time.Since(time.Now().Add(-timeout + time.Until(deadline)))
		fmt.Printf("[WebhookServer] Waiting for %s (%d/%d), elapsed: %ds\n",
			eventType, len(matching), count, int(elapsed.Seconds()))
		time.Sleep(pollInterval)
	}

	matching := s.GetNotificationsByType(eventType)
	fmt.Printf("[WebhookServer] Timeout waiting for %s: got %d/%d\n",
		eventType, len(matching), count)
	return matching
}

// PrintReport prints a summary report of all received notifications.
func (s *WebhookTestServer) PrintReport() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Println("\n=== Webhook Notification Report ===")
	fmt.Printf("Total notifications received: %d\n\n", len(s.notifications))

	if len(s.notifications) == 0 {
		fmt.Println("  (no notifications received)")
	}

	for _, n := range s.notifications {
		fmt.Printf("  [%s] %s (success=%v)\n",
			n.Timestamp.Format(time.RFC3339), n.EventType, n.HandlerSuccess)

		if result, ok := n.Parsed["result"].(map[string]interface{}); ok {
			if v, ok := result["orderStatus"]; ok {
				fmt.Printf("    orderStatus: %v\n", v)
			}
			if v, ok := result["subscriptionStatus"]; ok {
				fmt.Printf("    subscriptionStatus: %v\n", v)
			}
			if v, ok := result["refundStatus"]; ok {
				fmt.Printf("    refundStatus: %v\n", v)
			}
			if v, ok := result["acquiringOrderId"]; ok {
				fmt.Printf("    acquiringOrderId: %v\n", v)
			}
			if v, ok := result["subscriptionId"]; ok {
				fmt.Printf("    subscriptionId: %v\n", v)
			}
		}
	}

	fmt.Println("=== End Report ===")
}

func (s *WebhookTestServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	bodyStr := string(body)
	signature := r.Header.Get("X-SIGNATURE")

	fmt.Printf("[WebhookServer] Received webhook: %.200s...\n", bodyStr)

	// Parse event type
	eventType := "UNKNOWN"
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if et, ok := parsed["eventType"].(string); ok {
			eventType = et
		}
	}

	// Call SDK webhook handler
	result := s.webhookHandler.HandleWebhook(bodyStr, signature)

	// Store notification
	s.mu.Lock()
	s.notifications = append(s.notifications, ReceivedNotification{
		EventType:      eventType,
		Body:           bodyStr,
		Parsed:         parsed,
		Timestamp:      time.Now(),
		HandlerSuccess: result.Success,
		ResponseBody:   result.ResponseBody,
	})
	s.mu.Unlock()

	fmt.Printf("[WebhookServer] Processed %s: success=%v\n", eventType, result.Success)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SIGNATURE", result.ResponseSignature)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result.ResponseBody))
}

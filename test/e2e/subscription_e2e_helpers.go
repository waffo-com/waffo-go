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

func requireSubscriptionStatus(t *testing.T, subscriptionRequest, expectedStatus string, attempts int, interval time.Duration) *subscription.InquirySubscriptionData {
	t.Helper()

	params := &subscription.InquirySubscriptionParams{
		SubscriptionRequest: subscriptionRequest,
	}

	var finalStatus string
	var lastInquiry string
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := testWaffo.Subscription().Inquiry(context.Background(), params, nil)
		if err != nil {
			lastInquiry = fmt.Sprintf("error=%v", err)
		} else {
			lastInquiry = fmt.Sprintf("code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())
			t.Logf("Subscription inquiry: subscriptionRequest=%s, attempt=%d/%d, %s",
				subscriptionRequest, attempt, attempts, lastInquiry)
			if resp.IsSuccess() && resp.GetData() != nil {
				finalStatus = resp.GetData().SubscriptionStatus
				if finalStatus == expectedStatus {
					return resp.GetData()
				}
			}
		}

		t.Logf("Waiting for subscription status: subscriptionRequest=%s, expected=%s, attempt=%d/%d, current=%s",
			subscriptionRequest, expectedStatus, attempt, attempts, finalStatus)
		time.Sleep(interval)
	}

	t.Fatalf("Subscription did not reach expected status: subscriptionRequest=%s, expected=%s, finalStatus=%s, lastInquiry=%s",
		subscriptionRequest, expectedStatus, finalStatus, lastInquiry)
	return nil
}

func requireSubscriptionChangeStatus(t *testing.T, originSubscriptionRequest, subscriptionRequest, expectedStatus string, attempts int, interval time.Duration) *subscription.ChangeInquiryData {
	t.Helper()

	params := &subscription.ChangeInquiryParams{
		OriginSubscriptionRequest: originSubscriptionRequest,
		SubscriptionRequest:       subscriptionRequest,
	}

	var finalStatus string
	var lastInquiry string
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := testWaffo.Subscription().ChangeInquiry(context.Background(), params, nil)
		if err != nil {
			lastInquiry = fmt.Sprintf("error=%v", err)
		} else {
			lastInquiry = fmt.Sprintf("code=%s, msg=%s, data=%+v", resp.GetCode(), resp.GetMessage(), resp.GetData())
			t.Logf("Subscription change inquiry: originSubscriptionRequest=%s, subscriptionRequest=%s, attempt=%d/%d, %s",
				originSubscriptionRequest, subscriptionRequest, attempt, attempts, lastInquiry)
			if resp.IsSuccess() && resp.GetData() != nil {
				finalStatus = resp.GetData().SubscriptionChangeStatus
				if finalStatus == expectedStatus {
					return resp.GetData()
				}
			}
		}

		t.Logf("Waiting for subscription change status: originSubscriptionRequest=%s, subscriptionRequest=%s, expected=%s, attempt=%d/%d, current=%s",
			originSubscriptionRequest, subscriptionRequest, expectedStatus, attempt, attempts, finalStatus)
		time.Sleep(interval)
	}

	t.Fatalf("Subscription change did not reach expected status: originSubscriptionRequest=%s, subscriptionRequest=%s, expected=%s, finalStatus=%s, lastInquiry=%s",
		originSubscriptionRequest, subscriptionRequest, expectedStatus, finalStatus, lastInquiry)
	return nil
}

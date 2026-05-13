package subscription

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpdateSubscriptionParamsSerializesScheduledAmountPeriod(t *testing.T) {
	params := &UpdateSubscriptionParams{
		SubscriptionRequest: "sub_req_1",
		Amount:              "89.00",
		ProductInfo: &UpdateProductInfo{
			TrialPeriodAmount: "15.00",
			ScheduledAmounts: []ScheduledAmount{
				{Period: "2", Amount: "69.00"},
			},
		},
	}

	body, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal update params: %v", err)
	}

	payload := string(body)
	for _, expected := range []string{
		`"subscriptionRequest":"sub_req_1"`,
		`"amount":"89.00"`,
		`"trialPeriodAmount":"15.00"`,
		`"scheduledAmounts":[{"period":"2","amount":"69.00"}]`,
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("expected payload to contain %s, got %s", expected, payload)
		}
	}
	if strings.Contains(payload, "periodNumber") {
		t.Fatalf("update params must use period, not periodNumber: %s", payload)
	}
}

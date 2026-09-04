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
	if strings.Contains(payload, "topupInfo") {
		t.Fatalf("update params must omit empty topupInfo: %s", payload)
	}
}

func TestUpdateSubscriptionParamsSerializesTopupInfo(t *testing.T) {
	params := &UpdateSubscriptionParams{
		SubscriptionID: "SUB_UPDATE_001",
		TopupInfo: &SubscriptionUpdateTopupInfo{
			TopupRequest:   "mimo-topup-001",
			TopupAmount:    "30.00",
			Description:    "Seat add-on 3 seats",
			OrderExpiredAt: "2026-09-02T10:30:00+08:00",
		},
	}

	body, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal update params: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal update payload: %v", err)
	}
	topupInfo, ok := payload["topupInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("topupInfo missing from payload: %s", string(body))
	}
	if topupInfo["topupRequest"] != "mimo-topup-001" || topupInfo["topupAmount"] != "30.00" {
		t.Fatalf("unexpected topupInfo payload: %+v", topupInfo)
	}
	if topupInfo["description"] != "Seat add-on 3 seats" || topupInfo["orderExpiredAt"] != "2026-09-02T10:30:00+08:00" {
		t.Fatalf("unexpected topupInfo details: %+v", topupInfo)
	}
}

func TestUpdateSubscriptionDataDeserializesTopupInfo(t *testing.T) {
	payload := `{"subscriptionId":"SUB_UPDATE_001","topupInfo":{"acquiringOrderId":"A_TOPUP_001","topupRequest":"mimo-topup-001","topupAmount":"30.00","topupStatus":"AUTHORIZATION_REQUIRED","topupAction":"{\"webUrl\":\"https://checkout.example/topup\"}"}}`

	var data UpdateSubscriptionData
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}
	if data.TopupInfo == nil {
		t.Fatal("topupInfo missing from update response")
	}
	if data.TopupInfo.AcquiringOrderID != "A_TOPUP_001" || data.TopupInfo.TopupRequest != "mimo-topup-001" {
		t.Fatalf("unexpected topup identifiers: %+v", data.TopupInfo)
	}
	if data.TopupInfo.TopupAmount != "30.00" || data.TopupInfo.TopupStatus != "AUTHORIZATION_REQUIRED" {
		t.Fatalf("unexpected topup result: %+v", data.TopupInfo)
	}
	if !strings.Contains(data.TopupInfo.TopupAction, "webUrl") {
		t.Fatalf("unexpected topup action: %s", data.TopupInfo.TopupAction)
	}
}

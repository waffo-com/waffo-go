package order

import (
	"encoding/json"
	"testing"
)

func TestCreateOrderParamsNestedOrderFieldsAcceptMapValues(t *testing.T) {
	params := CreateOrderParams{
		AcqOrderExtSubscriptionInfo: map[string]interface{}{
			"subscriptionId": "sub_123",
			"periodNo":       2,
		},
		InnerCardData: map[string]interface{}{
			"cardBin":    "411111",
			"cardExpiry": "12/30",
		},
	}

	body, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	extInfo, ok := got["acqOrderExtSubscriptionInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected acqOrderExtSubscriptionInfo object, got %T", got["acqOrderExtSubscriptionInfo"])
	}
	if extInfo["subscriptionId"] != "sub_123" {
		t.Fatalf("expected subscriptionId=sub_123, got %v", extInfo["subscriptionId"])
	}

	cardData, ok := got["innerCardData"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected innerCardData object, got %T", got["innerCardData"])
	}
	if cardData["cardBin"] != "411111" {
		t.Fatalf("expected cardBin=411111, got %v", cardData["cardBin"])
	}
}

func TestCreateOrderParamsNestedOrderFieldsAcceptTypedValues(t *testing.T) {
	params := CreateOrderParams{
		AcqOrderExtSubscriptionInfo: &AcqOrderExtSubscriptionInfo{
			SubscriptionID:    "sub_456",
			PeriodNo:          3,
			MerchantRequest:   "req_456",
			SubscriptionEvent: "PERIOD_CHANGED",
		},
		InnerCardData: &WaffoTokenCardData{
			CardBin:    "555555",
			CardExpiry: "11/31",
			CardBinDataList: []CardBinData{
				{
					CardBin:    "555555",
					CardScheme: "MASTERCARD",
				},
			},
		},
	}

	body, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	extInfo, ok := got["acqOrderExtSubscriptionInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected acqOrderExtSubscriptionInfo object, got %T", got["acqOrderExtSubscriptionInfo"])
	}
	if extInfo["subscriptionEvent"] != "PERIOD_CHANGED" {
		t.Fatalf("expected subscriptionEvent=PERIOD_CHANGED, got %v", extInfo["subscriptionEvent"])
	}

	cardData, ok := got["innerCardData"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected innerCardData object, got %T", got["innerCardData"])
	}
	cardBins, ok := cardData["cardBinDataList"].([]interface{})
	if !ok || len(cardBins) != 1 {
		t.Fatalf("expected one cardBinDataList item, got %v", cardData["cardBinDataList"])
	}
}

package merchant

import (
	"encoding/json"
	"testing"
)

func TestPayMethodDetailUnmarshalContractFields(t *testing.T) {
	payload := []byte(`{
		"productName": "ONE_TIME_PAYMENT",
		"payMethodName": "GCASH",
		"country": "PHL",
		"payMethodType": "EWALLET",
		"currency": "PHP",
		"orderAmountMin": "1.00",
		"orderAmountMax": "500000.00",
		"refundAllowed": 21
	}`)

	var detail PayMethodDetail
	if err := json.Unmarshal(payload, &detail); err != nil {
		t.Fatalf("unmarshal PayMethodDetail: %v", err)
	}

	if detail.PayMethodType != "EWALLET" || detail.Currency != "PHP" {
		t.Fatalf("unexpected pay method classification: %+v", detail)
	}
	if detail.OrderAmountMin != "1.00" || detail.OrderAmountMax != "500000.00" {
		t.Fatalf("amount limits must retain string precision: %+v", detail)
	}
	if detail.RefundAllowed == nil || *detail.RefundAllowed != 21 {
		t.Fatalf("unexpected refundAllowed: got %v", detail.RefundAllowed)
	}
	if detail.CurrentStatus != "" {
		t.Fatalf("currentStatus should remain optional: %+v", detail)
	}
}

package order

import (
	"encoding/json"
	"testing"
)

func TestCreateOrderParamsX402InfoMarshals(t *testing.T) {
	params := CreateOrderParams{
		X402Info: &X402Info{
			X402Request:            true,
			PaymentSignatureHeader: "sig-header",
			X402Response:           "resp-payload",
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

	x402, ok := got["x402Info"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected x402Info object, got %T", got["x402Info"])
	}
	if x402["paymentSignatureHeader"] != "sig-header" {
		t.Fatalf("expected paymentSignatureHeader=sig-header, got %v", x402["paymentSignatureHeader"])
	}
	if x402["x402Request"] != true {
		t.Fatalf("expected x402Request=true, got %v", x402["x402Request"])
	}
}

func TestCreateOrderParamsPromotionInfoMarshals(t *testing.T) {
	params := CreateOrderParams{
		PromotionInfo: &PromotionInfo{
			PromotionCode: "SUMMER10",
			CouponID:      "coupon_123",
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

	promo, ok := got["promotionInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected promotionInfo object, got %T", got["promotionInfo"])
	}
	if promo["promotionCode"] != "SUMMER10" {
		t.Fatalf("expected promotionCode=SUMMER10, got %v", promo["promotionCode"])
	}
	if promo["couponId"] != "coupon_123" {
		t.Fatalf("expected couponId=coupon_123, got %v", promo["couponId"])
	}
}

func TestCreateOrderParamsLegacyFieldsRemainSourceCompatible(t *testing.T) {
	params := CreateOrderParams{
		ProductDetail: &ProductDetail{},
		AcqOrderExtSubscriptionInfo: &AcqOrderExtSubscriptionInfo{
			SubscriptionID: "sub_456",
		},
		ChannelID: "legacy-channel",
		InnerCardData: &WaffoTokenCardData{
			CardBin: "555555",
		},
	}

	payload, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if got["channelId"] != "legacy-channel" {
		t.Fatalf("expected legacy channelId, got %v", got["channelId"])
	}
	if _, ok := got["productDetail"].(map[string]interface{}); !ok {
		t.Fatalf("expected legacy productDetail object, got %T", got["productDetail"])
	}
}

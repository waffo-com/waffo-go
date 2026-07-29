package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/waffo-com/waffo-go/v2/config"
	"github.com/waffo-com/waffo-go/v2/core"
	waffonet "github.com/waffo-com/waffo-go/v2/net"
	"github.com/waffo-com/waffo-go/v2/types/subscription"
	"github.com/waffo-com/waffo-go/v2/utils"
)

type recordingSubscriptionTransport struct {
	request *waffonet.HttpRequest
}

func (t *recordingSubscriptionTransport) Send(ctx context.Context, req *waffonet.HttpRequest) (*waffonet.HttpResponse, error) {
	t.request = req
	return waffonet.NewHttpResponse(200, nil, []byte(`{"code":"0","msg":"success","data":{}}`)), nil
}

func newTestSubscriptionResourceWithTransport(t *testing.T, transport waffonet.HttpTransport) *SubscriptionResource {
	t.Helper()
	keyPair, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() failed: %v", err)
	}
	cfg := &config.WaffoConfig{
		APIKey:          "test-api-key",
		PrivateKey:      keyPair.PrivateKey,
		WaffoPublicKey:  keyPair.PublicKey,
		Environment:     config.Sandbox,
		MerchantID:      "MERCHANT_TEST_001",
		CustomTransport: transport,
	}
	return NewSubscriptionResource(core.NewWaffoHttpClient(cfg))
}

func TestSubscriptionCreatePostsWithoutSubscriptionManagementURL(t *testing.T) {
	transport := &recordingSubscriptionTransport{}
	resource := newTestSubscriptionResourceWithTransport(t, transport)

	resp, err := resource.Create(context.Background(), &subscription.CreateSubscriptionParams{
		SubscriptionRequest:    "sub_req_001",
		MerchantSubscriptionID: "merchant_sub_001",
		Currency:               "USD",
		Amount:                 "9.99",
		ProductInfo:            &subscription.ProductInfo{Description: "Premium", PeriodType: "MONTHLY", PeriodInterval: "1"},
		UserInfo:               &subscription.SubscriptionUserInfo{UserID: "user_001", UserEmail: "user@example.com"},
		PaymentInfo:            &subscription.SubscriptionPaymentInfo{ProductName: "SUBSCRIPTION"},
		NotifyURL:              "https://example.com/webhook",
	}, nil)

	if err != nil {
		t.Fatalf("expected create without subscriptionManagementUrl to post, got error: %v", err)
	}
	if resp == nil || !resp.IsSuccess() {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if transport.request == nil {
		t.Fatal("expected transport to receive a request")
	}
	if transport.request.Method != "POST" {
		t.Fatalf("expected POST request, got %s", transport.request.Method)
	}
	if transport.request.Body == nil {
		t.Fatal("expected request body")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(transport.request.Body, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if _, ok := body["subscriptionManagementUrl"]; ok {
		t.Fatalf("subscriptionManagementUrl should be omitted when unset, body=%s", string(transport.request.Body))
	}
}

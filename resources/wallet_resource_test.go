package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/waffo-com/waffo-go/v2/config"
	"github.com/waffo-com/waffo-go/v2/core"
	waffonet "github.com/waffo-com/waffo-go/v2/net"
	"github.com/waffo-com/waffo-go/v2/types/wallet"
	"github.com/waffo-com/waffo-go/v2/utils"
)

type walletResponseTransport struct {
	body string
}

func (t *walletResponseTransport) Send(_ context.Context, _ *waffonet.HttpRequest) (*waffonet.HttpResponse, error) {
	return waffonet.NewHttpResponse(200, nil, []byte(t.body)), nil
}

func TestWalletInquiryResponseParsing(t *testing.T) {
	keys, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	six, eighteen, zero := int32(6), int32(18), int32(0)
	tests := []struct {
		name string
		data string
		want wallet.WalletInquiryData
	}{
		{
			name: "payment options only",
			data: `{"paymentOptions":[{"network":"eip155:8453","asset":"asset-base","payTo":"wallet-base","decimals":6},{"network":"eip155:4217","asset":"asset-tempo","payTo":"wallet-tempo","decimals":18}]}`,
			want: wallet.WalletInquiryData{PaymentOptions: []wallet.PaymentOption{
				{Network: "eip155:8453", Asset: "asset-base", PayTo: "wallet-base", Decimals: &six},
				{Network: "eip155:4217", Asset: "asset-tempo", PayTo: "wallet-tempo", Decimals: &eighteen},
			}},
		},
		{name: "empty options", data: `{"paymentOptions":[]}`, want: wallet.WalletInquiryData{PaymentOptions: []wallet.PaymentOption{}}},
		{name: "missing options", data: `{}`, want: wallet.WalletInquiryData{}},
		{name: "null options", data: `{"paymentOptions":null}`, want: wallet.WalletInquiryData{}},
		{name: "null option fields", data: `{"paymentOptions":[{"network":null,"asset":null,"payTo":null,"decimals":null}]}`, want: wallet.WalletInquiryData{PaymentOptions: []wallet.PaymentOption{{}}}},
		{name: "zero decimals", data: `{"paymentOptions":[{"decimals":0}]}`, want: wallet.WalletInquiryData{PaymentOptions: []wallet.PaymentOption{{Decimals: &zero}}}},
		{
			name: "old fields do not synthesize options",
			data: `{"network":"legacy-network","asset":"legacy-asset","payTo":"legacy-wallet"}`,
			want: wallet.WalletInquiryData{},
		},
		{
			name: "unknown parent fields and optional decimals",
			data: `{"network":"legacy-network","asset":"legacy-asset","payTo":"legacy-wallet","paymentOptions":[{"network":"eip155:8453","asset":"asset-base","payTo":"wallet-base"}]}`,
			want: wallet.WalletInquiryData{
				PaymentOptions: []wallet.PaymentOption{{Network: "eip155:8453", Asset: "asset-base", PayTo: "wallet-base"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &walletResponseTransport{body: `{"code":"0","msg":"success","data":` + tt.data + `}`}
			client := core.NewWaffoHttpClient(&config.WaffoConfig{
				APIKey: "test-api-key", PrivateKey: keys.PrivateKey, WaffoPublicKey: keys.PublicKey,
				Environment: config.Sandbox, MerchantID: "wallet-test-merchant", CustomTransport: transport,
			})
			resp, err := NewWalletResource(client).Inquiry(context.Background(), &wallet.WalletInquiryParams{
				PaymentRequestID: "wallet-response-test", OrderCurrency: "USDC",
			}, nil)
			if err != nil || resp == nil || !resp.IsSuccess() || resp.GetData() == nil {
				t.Fatalf("wallet inquiry failed: err=%v, response=%+v", err, resp)
			}
			if !reflect.DeepEqual(*resp.GetData(), tt.want) {
				t.Fatalf("wallet data mismatch: got=%+v, want=%+v", *resp.GetData(), tt.want)
			}
		})
	}
}

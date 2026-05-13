package vectors

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/waffo-com/waffo-go/types/merchant"
	"github.com/waffo-com/waffo-go/types/order"
	"github.com/waffo-com/waffo-go/types/refund"
	"github.com/waffo-com/waffo-go/types/subscription"
)

// schemaFieldsFile is the test vector generated from openapi.json
type schemaFieldsFile struct {
	Schemas []schemaEntry `json:"schemas"`
}

type schemaEntry struct {
	OpenapiSchema    string   `json:"openapiSchema"`
	Fields           []string `json:"fields"`
	Required         []string `json:"required"`
	AllowExtraFields bool     `json:"allowExtraFields"`
	SdkTypes         struct {
		Go string `json:"go"`
	} `json:"sdkTypes"`
}

type jsonField struct {
	StructField string
	OmitEmpty   bool
}

// goTypeRegistry maps "package.TypeName" → reflect.Type
var goTypeRegistry = map[string]reflect.Type{
	// Order
	"order.CreateOrderParams":           reflect.TypeOf(order.CreateOrderParams{}),
	"order.AcqOrderExtSubscriptionInfo": reflect.TypeOf(order.AcqOrderExtSubscriptionInfo{}),
	"order.WaffoTokenCardData":          reflect.TypeOf(order.WaffoTokenCardData{}),
	"order.CreateOrderData":             reflect.TypeOf(order.CreateOrderData{}),
	"order.InquiryOrderParams":          reflect.TypeOf(order.InquiryOrderParams{}),
	"order.InquiryOrderData":            reflect.TypeOf(order.InquiryOrderData{}),
	"order.CancelOrderParams":           reflect.TypeOf(order.CancelOrderParams{}),
	"order.CancelOrderData":             reflect.TypeOf(order.CancelOrderData{}),
	"order.CaptureOrderParams":          reflect.TypeOf(order.CaptureOrderParams{}),
	"order.CaptureOrderData":            reflect.TypeOf(order.CaptureOrderData{}),
	"order.RefundOrderParams":           reflect.TypeOf(order.RefundOrderParams{}),
	"order.RefundOrderData":             reflect.TypeOf(order.RefundOrderData{}),
	// Refund
	"refund.InquiryRefundParams": reflect.TypeOf(refund.InquiryRefundParams{}),
	"refund.InquiryRefundData":   reflect.TypeOf(refund.InquiryRefundData{}),
	// Subscription
	"subscription.CreateSubscriptionParams":      reflect.TypeOf(subscription.CreateSubscriptionParams{}),
	"subscription.CreateSubscriptionData":        reflect.TypeOf(subscription.CreateSubscriptionData{}),
	"subscription.InquirySubscriptionParams":     reflect.TypeOf(subscription.InquirySubscriptionParams{}),
	"subscription.InquirySubscriptionData":       reflect.TypeOf(subscription.InquirySubscriptionData{}),
	"subscription.ProductInfo":                   reflect.TypeOf(subscription.ProductInfo{}),
	"subscription.PaymentDetail":                 reflect.TypeOf(subscription.PaymentDetail{}),
	"subscription.ManageSubscriptionParams":      reflect.TypeOf(subscription.ManageSubscriptionParams{}),
	"subscription.ManageSubscriptionData":        reflect.TypeOf(subscription.ManageSubscriptionData{}),
	"subscription.UpdateSubscriptionParams":      reflect.TypeOf(subscription.UpdateSubscriptionParams{}),
	"subscription.UpdateProductInfo":             reflect.TypeOf(subscription.UpdateProductInfo{}),
	"subscription.UpdateSubscriptionData":        reflect.TypeOf(subscription.UpdateSubscriptionData{}),
	"subscription.ScheduledAmount":               reflect.TypeOf(subscription.ScheduledAmount{}),
	"subscription.CancelSubscriptionParams":      reflect.TypeOf(subscription.CancelSubscriptionParams{}),
	"subscription.CancelSubscriptionData":        reflect.TypeOf(subscription.CancelSubscriptionData{}),
	"subscription.ChangeSubscriptionParams":      reflect.TypeOf(subscription.ChangeSubscriptionParams{}),
	"subscription.SubscriptionChangeProductInfo": reflect.TypeOf(subscription.SubscriptionChangeProductInfo{}),
	"subscription.ChangeSubscriptionData":        reflect.TypeOf(subscription.ChangeSubscriptionData{}),
	"subscription.ChangeInquiryParams":           reflect.TypeOf(subscription.ChangeInquiryParams{}),
	"subscription.ChangeInquiryData":             reflect.TypeOf(subscription.ChangeInquiryData{}),
	// Merchant
	"merchant.InquiryMerchantConfigParams": reflect.TypeOf(merchant.InquiryMerchantConfigParams{}),
	"merchant.InquiryMerchantConfigData":   reflect.TypeOf(merchant.InquiryMerchantConfigData{}),
}

// Required OpenAPI request fields that are intentionally omitted from the public
// zero-value payload because the SDK injects them before signing the request.
var autoInjectedRequiredFields = map[string]bool{
	"AcqOrderCreateRequest.merchantInfo":        true,
	"AcqOrderCreateRequest.orderRequestedAt":    true,
	"AcqOrderCancelRequest.merchantId":          true,
	"AcqOrderCancelRequest.orderRequestedAt":    true,
	"CaptureOrderRequest.merchantId":            true,
	"CaptureOrderRequest.captureRequestedAt":    true,
	"AcqOrderRefundRequest.merchantId":          true,
	"AcqOrderRefundRequest.requestedAt":         true,
	"AcqSubscriptionCreateRequest.merchantInfo": true,
	"AcqSubscriptionCreateRequest.requestedAt":  true,
	"SubscriptionCancelRequest.merchantId":      true,
	"SubscriptionCancelRequest.requestedAt":     true,
	"SubscriptionChangeRequest.merchantInfo":    true,
	"SubscriptionChangeRequest.requestedAt":     true,
}

// getJSONTags extracts all json tag names from a struct type.
// Returns a map of jsonTagName → fieldName for quick lookup.
func getJSONTags(t reflect.Type) map[string]jsonField {
	tags := make(map[string]jsonField)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		// Parse "fieldName,omitempty" → "fieldName"
		parts := strings.Split(tag, ",")
		jsonName := parts[0]
		omitEmpty := false
		for _, option := range parts[1:] {
			if option == "omitempty" {
				omitEmpty = true
				break
			}
		}
		tags[jsonName] = jsonField{StructField: field.Name, OmitEmpty: omitEmpty}
	}
	return tags
}

func validatesRequiredShape(schemaName string) bool {
	return strings.HasSuffix(schemaName, "Request") ||
		strings.HasSuffix(schemaName, "Request.ProductInfo") ||
		schemaName == "ProductInfo"
}

// TestSchemaFieldsAlignment validates that all Go SDK types have the fields
// defined in openapi.json (via schema-fields.json test vector).
func TestSchemaFieldsAlignment(t *testing.T) {
	data, err := os.ReadFile("../../../../sdk-spec/test-vectors/schema-fields.json")
	if err != nil {
		t.Fatalf("Failed to read schema-fields.json: %v", err)
	}

	var sf schemaFieldsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("Failed to parse schema-fields.json: %v", err)
	}

	for _, schema := range sf.Schemas {
		goTypeName := schema.SdkTypes.Go
		if goTypeName == "" {
			continue
		}

		t.Run(schema.OpenapiSchema+"→"+goTypeName, func(t *testing.T) {
			goType, ok := goTypeRegistry[goTypeName]
			if !ok {
				t.Fatalf("Go type %q not registered in goTypeRegistry", goTypeName)
			}

			jsonTags := getJSONTags(goType)

			// Check every openapi field exists as a json tag in the Go struct
			for _, field := range schema.Fields {
				if _, found := jsonTags[field]; !found {
					t.Errorf("MISSING field: openapi %q requires json tag %q, but Go type %s does not have it",
						schema.OpenapiSchema, field, goTypeName)
				}
			}

			if validatesRequiredShape(schema.OpenapiSchema) {
				for _, field := range schema.Required {
					info, found := jsonTags[field]
					if !found {
						continue
					}
					key := schema.OpenapiSchema + "." + field
					if info.OmitEmpty && !autoInjectedRequiredFields[key] {
						t.Errorf("REQUIRED field: openapi %q marks %q required, but Go type %s field %s uses omitempty",
							schema.OpenapiSchema, field, goTypeName, info.StructField)
					}
				}
			}

			// Check no unexpected json tags exist (excluding SDK infrastructure fields)
			excluded := map[string]bool{
				"extraParams": true,
				"metadata":    true,
			}
			openapiFields := make(map[string]bool)
			for _, f := range schema.Fields {
				openapiFields[f] = true
			}
			for jsonTag := range jsonTags {
				if excluded[jsonTag] {
					continue
				}
				if schema.AllowExtraFields {
					continue
				}
				if !openapiFields[jsonTag] {
					t.Errorf("EXTRA field: Go type %s has json tag %q which is not in openapi schema %q",
						goTypeName, jsonTag, schema.OpenapiSchema)
				}
			}
		})
	}
}

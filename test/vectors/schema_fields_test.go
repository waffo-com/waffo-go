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
	OpenapiSchema string   `json:"openapiSchema"`
	Fields        []string `json:"fields"`
	SdkTypes      struct {
		Go string `json:"go"`
	} `json:"sdkTypes"`
}

// goTypeRegistry maps "package.TypeName" → reflect.Type
var goTypeRegistry = map[string]reflect.Type{
	// Order
	"order.CreateOrderParams":  reflect.TypeOf(order.CreateOrderParams{}),
	"order.CreateOrderData":    reflect.TypeOf(order.CreateOrderData{}),
	"order.InquiryOrderParams": reflect.TypeOf(order.InquiryOrderParams{}),
	"order.InquiryOrderData":   reflect.TypeOf(order.InquiryOrderData{}),
	"order.CancelOrderParams":  reflect.TypeOf(order.CancelOrderParams{}),
	"order.CancelOrderData":    reflect.TypeOf(order.CancelOrderData{}),
	"order.RefundOrderParams":  reflect.TypeOf(order.RefundOrderParams{}),
	"order.RefundOrderData":    reflect.TypeOf(order.RefundOrderData{}),
	// Refund
	"refund.InquiryRefundParams": reflect.TypeOf(refund.InquiryRefundParams{}),
	"refund.InquiryRefundData":   reflect.TypeOf(refund.InquiryRefundData{}),
	// Subscription
	"subscription.CreateSubscriptionParams":  reflect.TypeOf(subscription.CreateSubscriptionParams{}),
	"subscription.CreateSubscriptionData":    reflect.TypeOf(subscription.CreateSubscriptionData{}),
	"subscription.InquirySubscriptionParams": reflect.TypeOf(subscription.InquirySubscriptionParams{}),
	"subscription.InquirySubscriptionData":   reflect.TypeOf(subscription.InquirySubscriptionData{}),
	"subscription.ManageSubscriptionParams":  reflect.TypeOf(subscription.ManageSubscriptionParams{}),
	"subscription.ManageSubscriptionData":    reflect.TypeOf(subscription.ManageSubscriptionData{}),
	"subscription.CancelSubscriptionParams":  reflect.TypeOf(subscription.CancelSubscriptionParams{}),
	"subscription.CancelSubscriptionData":    reflect.TypeOf(subscription.CancelSubscriptionData{}),
	"subscription.ChangeSubscriptionParams":  reflect.TypeOf(subscription.ChangeSubscriptionParams{}),
	"subscription.ChangeSubscriptionData":    reflect.TypeOf(subscription.ChangeSubscriptionData{}),
	"subscription.ChangeInquiryParams":       reflect.TypeOf(subscription.ChangeInquiryParams{}),
	"subscription.ChangeInquiryData":         reflect.TypeOf(subscription.ChangeInquiryData{}),
	// Merchant
	"merchant.InquiryMerchantConfigParams": reflect.TypeOf(merchant.InquiryMerchantConfigParams{}),
	"merchant.InquiryMerchantConfigData":   reflect.TypeOf(merchant.InquiryMerchantConfigData{}),
}

// getJSONTags extracts all json tag names from a struct type.
// Returns a map of jsonTagName → fieldName for quick lookup.
func getJSONTags(t reflect.Type) map[string]string {
	tags := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		// Parse "fieldName,omitempty" → "fieldName"
		jsonName := strings.Split(tag, ",")[0]
		tags[jsonName] = field.Name
	}
	return tags
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
				if !openapiFields[jsonTag] {
					t.Errorf("EXTRA field: Go type %s has json tag %q which is not in openapi schema %q",
						goTypeName, jsonTag, schema.OpenapiSchema)
				}
			}
		})
	}
}

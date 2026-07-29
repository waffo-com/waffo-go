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
	"github.com/waffo-com/waffo-go/types/wallet"
)

// schemaFieldsFile is the test vector generated from openapi.json
type schemaFieldsFile struct {
	Schemas []schemaEntry `json:"schemas"`
}

type schemaEntry struct {
	OpenapiSchema      string            `json:"openapiSchema"`
	Fields             []string          `json:"fields"`
	FieldTypes         map[string]string `json:"fieldTypes"`
	Required           []string          `json:"required"`
	AllowExtraFields   bool              `json:"allowExtraFields"`
	AllowedExtraFields []string          `json:"allowedExtraFields"`
	SdkTypes           struct {
		Go string `json:"go"`
	} `json:"sdkTypes"`
}

type jsonField struct {
	StructField string
	OmitEmpty   bool
	Kind        string
}

// acceptedKinds maps an openapi type kind to the Go kinds allowed for it.
// The Go SDK carries date-time values as strings.
var acceptedKinds = map[string][]string{
	"string":   {"string"},
	"datetime": {"string"},
	"integer":  {"integer"},
	"number":   {"number"},
	"boolean":  {"boolean"},
	"array":    {"array"},
	"object":   {"object"},
}

// goKind reduces a Go field type to a coarse kind comparable with openapi kinds.
func goKind(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "object"
	}
}

// goTypeRegistry maps "package.TypeName" → reflect.Type
var goTypeRegistry = map[string]reflect.Type{
	// Order
	"order.CreateOrderParams":  reflect.TypeOf(order.CreateOrderParams{}),
	"order.X402Info":           reflect.TypeOf(order.X402Info{}),
	"order.PromotionInfo":      reflect.TypeOf(order.PromotionInfo{}),
	"order.CreateOrderData":    reflect.TypeOf(order.CreateOrderData{}),
	"order.InquiryOrderParams": reflect.TypeOf(order.InquiryOrderParams{}),
	"order.InquiryOrderData":   reflect.TypeOf(order.InquiryOrderData{}),
	"order.CancelOrderParams":  reflect.TypeOf(order.CancelOrderParams{}),
	"order.CancelOrderData":    reflect.TypeOf(order.CancelOrderData{}),
	"order.CaptureOrderParams": reflect.TypeOf(order.CaptureOrderParams{}),
	"order.CaptureOrderData":   reflect.TypeOf(order.CaptureOrderData{}),
	"order.RefundOrderParams":  reflect.TypeOf(order.RefundOrderParams{}),
	"order.RefundOrderData":    reflect.TypeOf(order.RefundOrderData{}),
	"order.PaymentInfo":        reflect.TypeOf(order.PaymentInfo{}),
	// Refund
	"refund.InquiryRefundParams": reflect.TypeOf(refund.InquiryRefundParams{}),
	"refund.InquiryRefundData":   reflect.TypeOf(refund.InquiryRefundData{}),
	// Subscription
	"subscription.CreateSubscriptionParams":      reflect.TypeOf(subscription.CreateSubscriptionParams{}),
	"subscription.SubscriptionPromotionInfo":     reflect.TypeOf(subscription.SubscriptionPromotionInfo{}),
	"subscription.CreateSubscriptionData":        reflect.TypeOf(subscription.CreateSubscriptionData{}),
	"subscription.SubscriptionUserInfo":          reflect.TypeOf(subscription.SubscriptionUserInfo{}),
	"subscription.SubscriptionPaymentInfo":       reflect.TypeOf(subscription.SubscriptionPaymentInfo{}),
	"subscription.SubscriptionAddressInfo":       reflect.TypeOf(subscription.SubscriptionAddressInfo{}),
	"subscription.SubscriptionRiskData":          reflect.TypeOf(subscription.SubscriptionRiskData{}),
	"subscription.InquirySubscriptionParams":     reflect.TypeOf(subscription.InquirySubscriptionParams{}),
	"subscription.InquirySubscriptionData":       reflect.TypeOf(subscription.InquirySubscriptionData{}),
	"subscription.SubscriptionInfo":              reflect.TypeOf(subscription.SubscriptionInfo{}),
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
	"merchant.InquiryMerchantConfigParams":  reflect.TypeOf(merchant.InquiryMerchantConfigParams{}),
	"merchant.InquiryMerchantConfigData":    reflect.TypeOf(merchant.InquiryMerchantConfigData{}),
	"merchant.InquiryPayMethodConfigParams": reflect.TypeOf(merchant.InquiryPayMethodConfigParams{}),
	"merchant.InquiryPayMethodConfigData":   reflect.TypeOf(merchant.InquiryPayMethodConfigData{}),
	"merchant.PayMethodDetail":              reflect.TypeOf(merchant.PayMethodDetail{}),
	"merchant.FixedMaintenanceRule":         reflect.TypeOf(merchant.FixedMaintenanceRule{}),
	// Wallet
	"wallet.WalletInquiryParams": reflect.TypeOf(wallet.WalletInquiryParams{}),
	"wallet.WalletInquiryData":   reflect.TypeOf(wallet.WalletInquiryData{}),
}

// Required OpenAPI request fields that are intentionally omitted from the public
// zero-value payload because the SDK injects them before signing the request.
var autoInjectedRequiredFields = map[string]bool{
	"AcqOrderCreateRequest.merchantInfo":        true,
	"AcqOrderCreateRequest.orderRequestedAt":    true,
	"AcqOrderCancelRequest.merchantId":          true,
	"AcqOrderCancelRequest.orderRequestedAt":    true,
	"OrderCaptureRequest.merchantId":            true,
	"OrderCaptureRequest.captureRequestedAt":    true,
	"AcqOrderRefundRequest.merchantId":          true,
	"AcqOrderRefundRequest.requestedAt":         true,
	"AcqSubscriptionCreateRequest.merchantInfo": true,
	"AcqSubscriptionCreateRequest.requestedAt":  true,
	"SubscriptionCancelRequest.merchantId":      true,
	"SubscriptionCancelRequest.requestedAt":     true,
	"SubscriptionChangeRequest.merchantInfo":    true,
	"SubscriptionChangeRequest.requestedAt":     true,
	"WalletInquiryReq.merchantInfo":             true,
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
		tags[jsonName] = jsonField{StructField: field.Name, OmitEmpty: omitEmpty, Kind: goKind(field.Type)}
	}
	return tags
}

func containsKind(kinds []string, kind string) bool {
	for _, candidate := range kinds {
		if candidate == kind {
			return true
		}
	}
	return false
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

			// Check every openapi field is declared with a compatible Go type.
			// Field names alone cannot catch a scalar being declared as an object.
			for _, field := range schema.Fields {
				info, found := jsonTags[field]
				if !found {
					continue
				}
				openapiKind := schema.FieldTypes[field]
				accepted, known := acceptedKinds[openapiKind]
				if !known {
					continue
				}
				if !containsKind(accepted, info.Kind) {
					t.Errorf("TYPE MISMATCH: openapi %q field %q is %s, but Go type %s field %s is %s",
						schema.OpenapiSchema, field, openapiKind, goTypeName, info.StructField, info.Kind)
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
			allowedExtraFields := make(map[string]bool)
			for _, f := range schema.AllowedExtraFields {
				allowedExtraFields[f] = true
			}
			for jsonTag := range jsonTags {
				if excluded[jsonTag] {
					continue
				}
				if schema.AllowExtraFields {
					continue
				}
				if !openapiFields[jsonTag] && !allowedExtraFields[jsonTag] {
					t.Errorf("EXTRA field: Go type %s has json tag %q which is not in openapi schema %q",
						goTypeName, jsonTag, schema.OpenapiSchema)
				}
			}
		})
	}
}

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateSameCustomer_MatchingCustomerOK(t *testing.T) {
	var diags diag.Diagnostics
	validateSameCustomer(&diags, types.StringValue("123"), map[string]types.String{
		"campaign_id": types.StringValue("customers/123/campaigns/9"),
		"asset_id":    types.StringValue("customers/123/assets/7"),
	})
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
}

func TestValidateSameCustomer_FlagsMismatch(t *testing.T) {
	var diags diag.Diagnostics
	validateSameCustomer(&diags, types.StringValue("123"), map[string]types.String{
		"campaign_id": types.StringValue("customers/999/campaigns/9"),
	})
	if !diags.HasError() {
		t.Fatal("expected mismatch diagnostic")
	}
	if !contains(diags.Errors()[0].Detail(), "customer 999") {
		t.Errorf("diagnostic detail = %q", diags.Errors()[0].Detail())
	}
}

func TestValidateSameCustomer_FlagsEachMismatchSeparately(t *testing.T) {
	var diags diag.Diagnostics
	validateSameCustomer(&diags, types.StringValue("123"), map[string]types.String{
		"campaign_id": types.StringValue("customers/999/campaigns/9"),
		"asset_id":    types.StringValue("customers/888/assets/7"),
	})
	if got := len(diags.Errors()); got != 2 {
		t.Fatalf("want 2 errors, got %d: %+v", got, diags)
	}
}

func TestValidateSameCustomer_SkipsNullOrUnknown(t *testing.T) {
	var diags diag.Diagnostics
	validateSameCustomer(&diags, types.StringValue("123"), map[string]types.String{
		"campaign_id": types.StringNull(),
		"asset_id":    types.StringUnknown(),
	})
	if diags.HasError() {
		t.Fatalf("null/unknown should be skipped, got: %+v", diags)
	}
}

func TestValidateSameCustomer_SkipsWhenCustomerIDUnknown(t *testing.T) {
	var diags diag.Diagnostics
	validateSameCustomer(&diags, types.StringUnknown(), map[string]types.String{
		"campaign_id": types.StringValue("customers/999/campaigns/9"),
	})
	if diags.HasError() {
		t.Fatalf("unknown customer_id should skip validation, got: %+v", diags)
	}
}

func TestValidateSameCustomer_IgnoresMalformedRefs(t *testing.T) {
	var diags diag.Diagnostics
	validateSameCustomer(&diags, types.StringValue("123"), map[string]types.String{
		"campaign_id": types.StringValue("not-a-resource-name"),
	})
	if diags.HasError() {
		t.Fatalf("malformed ref should be left to other validators, got: %+v", diags)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

package provider

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// runesLengthAtMost limits a string by Unicode character (rune) count rather
// than byte length. The Google Ads API's documented limits — 30 chars for
// headlines, 90 for descriptions, 15 for paths — count characters, not
// bytes, so the framework's stringvalidator.LengthAtMost (byte-based)
// rejects valid input like "Experiência" (11 chars / 13 bytes).
func runesLengthAtMost(max int) validator.String {
	return runesLengthValidator{max: max}
}

type runesLengthValidator struct {
	max int
}

func (v runesLengthValidator) Description(_ context.Context) string {
	return fmt.Sprintf("string length must be at most %d Unicode characters", v.max)
}

func (v runesLengthValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v runesLengthValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	n := utf8.RuneCountInString(s)
	if n > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value Length",
			fmt.Sprintf("Attribute %s string length must be at most %d Unicode characters, got: %d", req.Path, v.max, n),
		)
	}
}

// validateSameCustomer asserts that the `customer_id` segment of each
// resource-name reference matches the resource's own customer_id. The
// Ads API rejects cross-customer references at apply time with an
// opaque error; surfacing it at plan time saves a round trip and points
// at the exact attribute.
//
// `refs` maps the attribute name (used to build the diagnostic path) to
// the Terraform value holding the parent resource name. Null/unknown
// values are skipped — the framework will surface those separately.
func validateSameCustomer(diags *diag.Diagnostics, customerID types.String, refs map[string]types.String) {
	if customerID.IsNull() || customerID.IsUnknown() || customerID.ValueString() == "" {
		return
	}
	expected := customerID.ValueString()
	for attr, ref := range refs {
		if ref.IsNull() || ref.IsUnknown() || ref.ValueString() == "" {
			continue
		}
		got, err := googleads.CustomerOfResourceName(ref.ValueString())
		if err != nil {
			continue
		}
		if got != expected {
			diags.AddAttributeError(
				path.Root(attr),
				"Resource references a different customer",
				fmt.Sprintf("%s belongs to customer %s but customer_id is %s. Cross-customer references are rejected by the Google Ads API.", attr, got, expected),
			)
		}
	}
}

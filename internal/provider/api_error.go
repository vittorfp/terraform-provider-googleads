package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// addAPIErrorDiagnostics expands an error from the Google Ads API into
// one Terraform diagnostic per failed field. Non-*APIError values fall
// through to a single AddError so call sites don't need to type-assert.
//
// When the API reports a field path that maps to a known HCL attribute
// (the leaf segment matches an entry in `attrMap`), the diagnostic is
// attached to that attribute path, so the Terraform UI highlights the
// offending line. Otherwise it lands as a top-level error with the
// raw field path in the detail.
func addAPIErrorDiagnostics(diags *diag.Diagnostics, summary string, attrMap map[string]path.Path, err error) {
	var apiErr *googleads.APIError
	if !errors.As(err, &apiErr) {
		diags.AddError(summary, err.Error())
		return
	}
	if len(apiErr.Details) == 0 {
		diags.AddError(summary, apiErr.Error())
		return
	}
	for _, d := range apiErr.Details {
		detail := d.Message
		if d.FieldPath != "" {
			detail = fmt.Sprintf("%s (API field path: %s)", d.Message, d.FieldPath)
		}
		if p, ok := attrMap[d.LeafField]; ok {
			diags.AddAttributeError(p, summary, detail)
		} else {
			diags.AddError(summary, detail)
		}
	}
}

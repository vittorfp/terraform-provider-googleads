package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

func TestAddAPIErrorDiagnostics_PlainError(t *testing.T) {
	var diags diag.Diagnostics
	addAPIErrorDiagnostics(&diags, "create budget", nil, errFromString("boom"))
	if got := len(diags.Errors()); got != 1 {
		t.Fatalf("len(errors) = %d, want 1", got)
	}
	if got := diags.Errors()[0].Summary(); got != "create budget" {
		t.Errorf("summary = %q", got)
	}
}

func TestAddAPIErrorDiagnostics_APIErrorNoDetails(t *testing.T) {
	var diags diag.Diagnostics
	apiErr := &googleads.APIError{Code: "INVALID_ARGUMENT", Message: "bad input"}
	addAPIErrorDiagnostics(&diags, "create", nil, apiErr)
	if got := len(diags.Errors()); got != 1 {
		t.Fatalf("len(errors) = %d", got)
	}
}

func TestAddAPIErrorDiagnostics_ExpandsPerDetail(t *testing.T) {
	var diags diag.Diagnostics
	apiErr := &googleads.APIError{
		Code:    "INVALID_ARGUMENT",
		Message: "bad input",
		Details: []googleads.APIErrorDetail{
			{FieldPath: "operations[0].create.name", LeafField: "name", Message: "name required"},
			{FieldPath: "operations[0].create.amount_micros", LeafField: "amount_micros", Message: "must be positive"},
		},
	}
	attrMap := map[string]path.Path{
		"name":          path.Root("name"),
		"amount_micros": path.Root("amount_micros"),
	}
	addAPIErrorDiagnostics(&diags, "create", attrMap, apiErr)

	if got := len(diags.Errors()); got != 2 {
		t.Fatalf("len(errors) = %d, want 2", got)
	}
	// Each diagnostic carries the API field path in the detail line.
	for _, d := range diags.Errors() {
		if !strings.Contains(d.Detail(), "API field path:") {
			t.Errorf("detail missing API field path: %q", d.Detail())
		}
	}
}

func TestAddAPIErrorDiagnostics_UnknownFieldFallsToTopLevel(t *testing.T) {
	var diags diag.Diagnostics
	apiErr := &googleads.APIError{
		Code: "INVALID_ARGUMENT",
		Details: []googleads.APIErrorDetail{
			{FieldPath: "operations[0].create.weird_field", LeafField: "weird_field", Message: "unknown"},
		},
	}
	// attrMap empty — the detail's LeafField isn't in there.
	addAPIErrorDiagnostics(&diags, "create", map[string]path.Path{}, apiErr)
	if got := len(diags.Errors()); got != 1 {
		t.Fatalf("len(errors) = %d", got)
	}
	if !strings.Contains(diags.Errors()[0].Detail(), "weird_field") {
		t.Errorf("detail missing field path: %q", diags.Errors()[0].Detail())
	}
}

type stringErr struct{ s string }

func (e *stringErr) Error() string { return e.s }
func errFromString(s string) error { return &stringErr{s: s} }

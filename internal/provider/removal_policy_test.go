package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnforceRemovalPolicy_SoftAllows(t *testing.T) {
	var diags diag.Diagnostics
	if !enforceRemovalPolicy(&diags, types.StringValue("soft"), "campaign") {
		t.Fatal("soft should allow destroy")
	}
	if diags.HasError() {
		t.Fatalf("soft should not emit error, got: %+v", diags)
	}
}

func TestEnforceRemovalPolicy_NullAllows(t *testing.T) {
	var diags diag.Diagnostics
	if !enforceRemovalPolicy(&diags, types.StringNull(), "campaign") {
		t.Fatal("null (treated as soft) should allow destroy")
	}
}

func TestEnforceRemovalPolicy_ProtectBlocks(t *testing.T) {
	var diags diag.Diagnostics
	if enforceRemovalPolicy(&diags, types.StringValue("protect"), "campaign") {
		t.Fatal("protect should block destroy")
	}
	if !diags.HasError() {
		t.Fatal("protect should emit error diagnostic")
	}
	d := diags.Errors()[0]
	if d.Summary() != "Destroy blocked by removal_policy = \"protect\"" {
		t.Errorf("summary = %q", d.Summary())
	}
}


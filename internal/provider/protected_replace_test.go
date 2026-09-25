package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// rawWithValue returns a tftypes.Value-backed tfsdk.State (or Plan)
// that is non-null at the root, so the modifier treats it as an
// existing resource. Only the root null-ness matters for our checks.
func nonNullRoot() tfsdk.State {
	v := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"x": tftypes.String},
	}, map[string]tftypes.Value{"x": tftypes.NewValue(tftypes.String, "v")})
	return tfsdk.State{Raw: v}
}

func nullRoot() tfsdk.State {
	v := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"x": tftypes.String},
	}, nil)
	return tfsdk.State{Raw: v}
}

func runModifier(t *testing.T, stateVal, planVal types.String, stateNull, planNull bool) (*planmodifier.StringResponse, bool) {
	t.Helper()
	state := nonNullRoot()
	if stateNull {
		state = nullRoot()
	}
	plan := tfsdk.Plan{Raw: nonNullRoot().Raw}
	if planNull {
		plan = tfsdk.Plan{Raw: nullRoot().Raw}
	}
	req := planmodifier.StringRequest{
		Path:       path.Root("field"),
		StateValue: stateVal,
		PlanValue:  planVal,
		State:      state,
		Plan:       plan,
	}
	resp := &planmodifier.StringResponse{}
	ProtectedReplace().PlanModifyString(context.Background(), req, resp)
	return resp, resp.Diagnostics.HasError()
}

func TestProtectedReplace_CreateIsNoOp(t *testing.T) {
	allowDestructiveReplace.Store(false)
	resp, hadErr := runModifier(t, types.StringNull(), types.StringValue("SEARCH"), true, false)
	if hadErr {
		t.Fatalf("create should be no-op, got diags: %+v", resp.Diagnostics)
	}
	if resp.RequiresReplace {
		t.Fatal("create should not set RequiresReplace")
	}
}

func TestProtectedReplace_DestroyIsNoOp(t *testing.T) {
	allowDestructiveReplace.Store(false)
	resp, hadErr := runModifier(t, types.StringValue("SEARCH"), types.StringNull(), false, true)
	if hadErr {
		t.Fatalf("destroy should be no-op, got diags: %+v", resp.Diagnostics)
	}
}

func TestProtectedReplace_EqualValuesIsNoOp(t *testing.T) {
	allowDestructiveReplace.Store(false)
	resp, hadErr := runModifier(t, types.StringValue("SEARCH"), types.StringValue("SEARCH"), false, false)
	if hadErr {
		t.Fatalf("no change should be no-op, got: %+v", resp.Diagnostics)
	}
	if resp.RequiresReplace {
		t.Fatal("equal values should not set RequiresReplace")
	}
}

func TestProtectedReplace_BlocksChangeByDefault(t *testing.T) {
	allowDestructiveReplace.Store(false)
	resp, hadErr := runModifier(t, types.StringValue("SEARCH"), types.StringValue("DISPLAY"), false, false)
	if !hadErr {
		t.Fatal("expected error diagnostic blocking the change")
	}
	if resp.RequiresReplace {
		t.Fatal("must not request replace when blocking — Terraform should not destroy anything")
	}
	d := resp.Diagnostics.Errors()[0]
	if d.Summary() != "Protected immutable field changed" {
		t.Errorf("summary = %q", d.Summary())
	}
}

func TestProtectedReplace_AllowsChangeWhenFlagSet(t *testing.T) {
	allowDestructiveReplace.Store(true)
	defer allowDestructiveReplace.Store(false)
	resp, hadErr := runModifier(t, types.StringValue("SEARCH"), types.StringValue("DISPLAY"), false, false)
	if hadErr {
		t.Fatalf("flag-on should allow change, got diags: %+v", resp.Diagnostics)
	}
	if !resp.RequiresReplace {
		t.Fatal("flag-on should fall back to RequiresReplace semantics")
	}
}

func TestProtectedReplace_UnknownPlanIsNoOp(t *testing.T) {
	allowDestructiveReplace.Store(false)
	resp, hadErr := runModifier(t, types.StringValue("SEARCH"), types.StringUnknown(), false, false)
	if hadErr {
		t.Fatalf("unknown plan should be no-op (resolved at apply), got: %+v", resp.Diagnostics)
	}
}

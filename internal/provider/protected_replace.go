package provider

import (
	"context"
	"sync/atomic"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// allowDestructiveReplace is the package-level escape hatch read by
// ProtectedReplace plan modifiers. The provider's Configure step writes
// it from the `allow_destructive_replace` attribute (or
// GOOGLE_ADS_ALLOW_DESTRUCTIVE_REPLACE env var). Default — false — makes
// any change to a protected field error at plan time, preserving
// Google-Ads-side state like smart-bidding learning.
//
// Plan modifiers in terraform-plugin-framework don't receive provider
// data directly, hence the package-level atomic. The trade-off: if a
// user runs multiple provider configurations in one process, the last
// Configure call wins for everyone. That's fine for the common case
// and an acceptable simplification for now.
var allowDestructiveReplace atomic.Bool

// ProtectedReplace is a string plan modifier that errors at plan time
// when the value would change on an existing resource — instead of
// silently triggering a destroy+recreate that would lose Google
// Ads-side accumulated state (smart bidding learning, performance
// history, etc.).
//
// To override (e.g. when intentionally migrating off a strategy), set
// the provider's `allow_destructive_replace = true` once for the run.
//
// On create (state null) or destroy (plan null) the modifier is a
// no-op. Unknown plan values are also passed through — Terraform
// resolves those on apply.
func ProtectedReplace() planmodifier.String {
	return protectedReplaceModifier{}
}

type protectedReplaceModifier struct{}

func (protectedReplaceModifier) Description(_ context.Context) string {
	return "Changes to this field would force the resource to be destroyed and recreated, which loses Google Ads-side accumulated state. The provider blocks the change at plan time. Set the provider's allow_destructive_replace = true to override."
}

func (m protectedReplaceModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (protectedReplaceModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Create or destroy — let it through.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.Equal(req.PlanValue) {
		return
	}

	if allowDestructiveReplace.Load() {
		// Operator opted in — fall back to standard replace semantics.
		resp.RequiresReplace = true
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Protected immutable field changed",
		"Changing "+req.Path.String()+" from "+req.StateValue.ValueString()+" to "+req.PlanValue.ValueString()+
			" would destroy and recreate this resource, losing Google Ads-side state (smart bidding learning, performance history). "+
			"Set the provider's allow_destructive_replace = true to override, or revert the change.",
	)
}

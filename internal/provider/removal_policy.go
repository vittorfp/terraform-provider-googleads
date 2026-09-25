package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Removal policies. `soft` (default) is the historical behavior: a
// Terraform destroy calls the Ads API Remove mutation, which marks the
// resource REMOVED in Ads (it's never hard-deleted on Google's side).
// `protect` is the safeguard: a destroy is refused at apply time,
// forcing the operator to either flip the policy back to soft or
// delete out-of-band first.
const (
	RemovalPolicySoft    = "soft"
	RemovalPolicyProtect = "protect"
)

// removalPolicyAttribute builds the schema attribute. Embed in any
// resource's Attributes map under the key "removal_policy" — keeping
// the spelling consistent across resources matters for tooling.
//
// Optional, not Computed: this attribute has no API representation, so
// there's nothing to read back during refresh/import. Leaving it null
// in state when the user didn't set it (rather than auto-defaulting to
// "soft") means `terraform import` followed by `terraform plan` is a
// no-op when the user's HCL also omits the field — instead of
// surfacing a "(known after apply) -> soft" diff (issue #36).
//
// enforceRemovalPolicy below treats a null/empty value as "soft", so
// the runtime semantics match the historical default.
func removalPolicyAttribute(kindDesc string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: "Controls what happens when Terraform plans to destroy this " + kindDesc + ". `soft` (the default behavior when unset) calls the Ads API Remove mutation, marking it REMOVED in Ads. `protect` blocks the destroy at apply time — useful for production resources where an accidental `terraform destroy` (or removal from config) shouldn't be allowed to take them offline. To destroy a protected resource, flip the policy to `soft` and apply, or remove it manually in the Ads UI and `terraform state rm` it.",
		Validators: []validator.String{
			stringvalidator.OneOf(RemovalPolicySoft, RemovalPolicyProtect),
		},
	}
}

// enforceRemovalPolicy emits a blocking diagnostic when the policy is
// `protect`, in which case the caller must skip the actual destroy
// mutation. Returns true if the destroy may proceed (policy is soft or
// unset).
func enforceRemovalPolicy(diags *diag.Diagnostics, policy types.String, kindDesc string) bool {
	if policy.ValueString() != RemovalPolicyProtect {
		return true
	}
	diags.AddError(
		"Destroy blocked by removal_policy = \"protect\"",
		"This "+kindDesc+" has removal_policy set to \"protect\", so Terraform will not call the Ads API Remove mutation. "+
			"To destroy it: change removal_policy to \"soft\" and apply, or remove it in the Google Ads UI and `terraform state rm` it.",
	)
	return false
}


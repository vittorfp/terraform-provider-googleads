package provider

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestComputedAttributesUseStateForUnknown locks in fix #36 problem 1.
// Every Optional+Computed attribute on every resource must declare a
// UseStateForUnknown plan modifier. Without it, the framework
// reports the attribute as "(known after apply)" on every plan
// after the first apply / import — Terraform sees that as a diff
// and proposes a no-op change every time. Users see perpetual
// "1 to change, in-place" plans that flap a few computed values.
//
// Exception: `id` is Computed-only and already uses
// UseStateForUnknown explicitly. We don't enforce this on
// Required attributes (those have no Computed semantics) or on
// Computed-only attributes whose source of truth is the API
// (those are intentionally re-read on every refresh, e.g. labels'
// resource_name suffix when the API assigns it).
//
// The list of resources audited mirrors status_no_replace_test —
// the 15 most-imported resources where a perpetual diff hurts.
func TestComputedAttributesUseStateForUnknown(t *testing.T) {
	cases := []struct {
		name string
		ctor func() resource.Resource
		// skip lists attribute names that are Computed-without-UseStateForUnknown
		// on purpose. Be very specific about why each one is here.
		skip map[string]string
	}{
		{"campaign_budget", NewCampaignBudgetResource, nil},
		{"campaign", NewCampaignResource, nil},
		{"ad_group", NewAdGroupResource, nil},
		{"ad_group_ad", NewAdGroupAdResource, nil},
		{"ad_group_criterion", NewAdGroupCriterionResource, nil},
		{"ad_group_audience_criterion", NewAdGroupAudienceCriterionResource, nil},
		{"campaign_criterion", NewCampaignCriterionResource, nil},
		{"campaign_asset", NewCampaignAssetResource, nil},
		{"ad_group_asset", NewAdGroupAssetResource, nil},
		{"customer_asset", NewCustomerAssetResource, nil},
		{"asset_group", NewAssetGroupResource, nil},
		{"asset_group_asset", NewAssetGroupAssetResource, nil},
		{"shared_set", NewSharedSetResource, nil},
		{"shared_criterion", NewSharedCriterionResource, nil},
		{"campaign_shared_set", NewCampaignSharedSetResource, nil},
		{"conversion_action", NewConversionActionResource, nil},
		{"label", NewLabelResource, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp resource.SchemaResponse
			tc.ctor().Schema(context.Background(), resource.SchemaRequest{}, &resp)
			for name, attr := range resp.Schema.Attributes {
				if reason, skipped := tc.skip[name]; skipped {
					t.Logf("skipping %s.%s: %s", tc.name, name, reason)
					continue
				}
				if !isComputed(attr) {
					continue
				}
				if name == "id" {
					// id always uses UseStateForUnknown; the check below also
					// catches it, but spell it out so future readers know it's
					// not exempt.
				}
				if !hasUseStateForUnknown(attr) {
					t.Errorf("%s.%s is Computed without UseStateForUnknown — terraform will report `(known after apply)` on every plan after the first apply/import. Add UseStateForUnknown to its PlanModifiers.", tc.name, name)
				}
			}
		})
	}
}

func isComputed(attr resourceschema.Attribute) bool {
	type computedAttr interface{ IsComputed() bool }
	if c, ok := attr.(computedAttr); ok {
		return c.IsComputed()
	}
	return false
}

func hasUseStateForUnknown(attr resourceschema.Attribute) bool {
	// PlanModifiers field has a kind-specific type (StringPlanModifiers,
	// BoolPlanModifiers, etc.), so reflect.
	v := reflect.ValueOf(attr)
	f := v.FieldByName("PlanModifiers")
	if !f.IsValid() {
		return false
	}
	if f.Kind() != reflect.Slice {
		return false
	}
	for i := 0; i < f.Len(); i++ {
		name := reflect.TypeOf(f.Index(i).Interface()).String()
		if strings.Contains(strings.ToLower(name), "usestateforunknown") {
			return true
		}
	}
	return false
}

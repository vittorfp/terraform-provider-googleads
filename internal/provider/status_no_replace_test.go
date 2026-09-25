package provider

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestStatusIsNotReplaceTrigger walks every resource that exposes a
// `status` attribute and asserts none of them mark it as
// replace-triggering. The Ads API rules out a destructive recreate as
// a way to flip a campaign / ad group / criterion / ad between
// ENABLED, PAUSED, and REMOVED — those are all live mutations. If a
// PR ever wires `RequiresReplace()` onto a status field, this test
// catches it before it ships and silently torpedoes Smart Bidding
// learning on the next plan that toggles status.
func TestStatusIsNotReplaceTrigger(t *testing.T) {
	cases := []struct {
		name string
		ctor func() resource.Resource
	}{
		{"campaign", NewCampaignResource},
		{"ad_group", NewAdGroupResource},
		{"ad_group_ad", NewAdGroupAdResource},
		{"ad_group_criterion", NewAdGroupCriterionResource},
		{"ad_group_audience_criterion", NewAdGroupAudienceCriterionResource},
		{"campaign_criterion", NewCampaignCriterionResource},
		{"campaign_asset", NewCampaignAssetResource},
		{"ad_group_asset", NewAdGroupAssetResource},
		{"customer_asset", NewCustomerAssetResource},
		{"asset_group", NewAssetGroupResource},
		{"asset_group_asset", NewAssetGroupAssetResource},
		{"campaign_shared_set", NewCampaignSharedSetResource},
		{"shared_set", NewSharedSetResource},
		{"conversion_action", NewConversionActionResource},
		{"label", NewLabelResource},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp resource.SchemaResponse
			tc.ctor().Schema(context.Background(), resource.SchemaRequest{}, &resp)

			attr, ok := resp.Schema.Attributes["status"]
			if !ok {
				t.Fatalf("%s: no status attribute found", tc.name)
			}
			sa, ok := attr.(resourceschema.StringAttribute)
			if !ok {
				t.Fatalf("%s: status is %T, want StringAttribute", tc.name, attr)
			}
			for _, pm := range sa.PlanModifiers {
				name := reflect.TypeOf(pm).String()
				if strings.Contains(strings.ToLower(name), "requiresreplace") {
					t.Errorf("%s.status has a replace-triggering plan modifier (%s) — status toggles must be mutations, never destroy+recreate (Smart Bidding learning loss)", tc.name, name)
				}
			}
		})
	}
}

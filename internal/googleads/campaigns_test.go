package googleads

import (
	"context"
	"strings"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/resources"
)

func TestCreateCampaign_PicksTheRightBiddingOneof(t *testing.T) {
	cases := []struct {
		name             string
		strategy         string
		wantManualCpc    bool
		wantTargetCpa    int64
		wantTargetRoas   float64
		wantBidCeilMicro int64
	}{
		{"default-manual-cpc", "", true, 0, 0, 0},
		{"explicit-manual-cpc", "MANUAL_CPC", true, 0, 0, 0},
		{"target-cpa-with-params", "TARGET_CPA", false, 1_000_000, 0, 2_000_000},
		{"target-roas-with-params", "TARGET_ROAS", false, 0, 3.5, 5_000_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t)
			c := ts.newClient(t)
			in := CampaignInput{
				CustomerID:             "123",
				Name:                   "tf-test",
				AdvertisingChannelType: "SEARCH",
				CampaignBudget:         "customers/123/campaignBudgets/1",
				BiddingStrategyType:    tc.strategy,
			}
			if tc.wantTargetCpa != 0 {
				v := tc.wantTargetCpa
				in.TargetCpaMicros = &v
			}
			if tc.wantTargetRoas != 0 {
				v := tc.wantTargetRoas
				in.TargetRoas = &v
			}
			if tc.wantBidCeilMicro != 0 {
				v := tc.wantBidCeilMicro
				in.CpcBidCeilingMicros = &v
			}
			if _, err := c.CreateCampaign(context.Background(), in); err != nil {
				t.Fatalf("CreateCampaign: %v", err)
			}
			create := ts.campaignOps[0].GetCreate()
			if create == nil {
				t.Fatal("not a Create")
			}
			switch s := create.GetCampaignBiddingStrategy().(type) {
			case *resources.Campaign_ManualCpc:
				if !tc.wantManualCpc {
					t.Errorf("got ManualCpc, expected another strategy")
				}
				_ = s
			case *resources.Campaign_TargetCpa:
				if tc.wantManualCpc {
					t.Errorf("got TargetCpa, expected ManualCpc")
				}
				if got := s.TargetCpa.GetTargetCpaMicros(); got != tc.wantTargetCpa {
					t.Errorf("TargetCpaMicros = %d", got)
				}
				if got := s.TargetCpa.GetCpcBidCeilingMicros(); got != tc.wantBidCeilMicro {
					t.Errorf("CpcBidCeilingMicros = %d", got)
				}
			case *resources.Campaign_TargetRoas:
				if tc.wantManualCpc {
					t.Errorf("got TargetRoas, expected ManualCpc")
				}
				if got := s.TargetRoas.GetTargetRoas(); got != tc.wantTargetRoas {
					t.Errorf("TargetRoas = %g", got)
				}
				if got := s.TargetRoas.GetCpcBidCeilingMicros(); got != tc.wantBidCeilMicro {
					t.Errorf("CpcBidCeilingMicros = %d", got)
				}
			default:
				t.Fatalf("unexpected strategy oneof: %T", s)
			}
		})
	}
}

func TestUpdateCampaign_RebuildsStrategyOneofForParamPaths(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/campaigns/42"
	in := CampaignInput{
		BiddingStrategyType: "TARGET_CPA",
	}
	v := int64(7_500_000)
	in.TargetCpaMicros = &v
	// Changing target_cpa.target_cpa_micros requires the parent oneof
	// to be present on the request, even though only the child field
	// is named in the mask.
	if err := c.UpdateCampaign(context.Background(), rn, in, []string{"target_cpa.target_cpa_micros"}); err != nil {
		t.Fatalf("UpdateCampaign: %v", err)
	}
	update := ts.campaignOps[0].GetUpdate()
	if update == nil {
		t.Fatal("not an Update")
	}
	tc, ok := update.GetCampaignBiddingStrategy().(*resources.Campaign_TargetCpa)
	if !ok {
		t.Fatalf("oneof should be TargetCpa, got %T", update.GetCampaignBiddingStrategy())
	}
	if got := tc.TargetCpa.GetTargetCpaMicros(); got != 7_500_000 {
		t.Errorf("TargetCpaMicros = %d", got)
	}
}

func TestRemoveCampaign_SendsResourceName(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/campaigns/42"
	if err := c.RemoveCampaign(context.Background(), rn); err != nil {
		t.Fatalf("RemoveCampaign: %v", err)
	}
	if got := ts.campaignOps[0].GetRemove(); got != rn {
		t.Errorf("Remove = %q", got)
	}
}

// Compile-time silencer for unused imports in helper packages that some
// downstream tests may reference. Cheap insurance.
var _ = common.ManualCpc{}

// TestCampaignSelectClause_DropsUnselectableFields locks in the fix
// for issue #35: the Ads API rejects these two paths in GAQL
// SELECT, and a future "let me add this for completeness" edit
// would silently break the read path for every campaign in every
// account until someone runs tfgen or terraform import.
func TestCampaignSelectClause_DropsUnselectableFields(t *testing.T) {
	forbidden := []string{
		"campaign.maximize_conversions.cpc_bid_ceiling_micros",
		"campaign.maximize_conversion_value.cpc_bid_ceiling_micros",
	}
	for _, f := range forbidden {
		if strings.Contains(campaignSelectClause, f) {
			t.Errorf("campaignSelectClause must not contain %q — API rejects it in SELECT (issue #35)", f)
		}
	}
}

// TestCampaignSelectClause_HasTargetSpendCeiling locks in the fix
// for issue #39: TARGET_SPEND ("Maximize Clicks" in the UI) is one
// of the most common bidding strategies on search accounts. The
// per-click cap lives on campaign.target_spend.cpc_bid_ceiling_micros;
// without it in SELECT, every TARGET_SPEND campaign reads back
// cpc_bid_ceiling_micros = 0 and Terraform reports perpetual drift.
func TestCampaignSelectClause_HasTargetSpendCeiling(t *testing.T) {
	want := "campaign.target_spend.cpc_bid_ceiling_micros"
	if !strings.Contains(campaignSelectClause, want) {
		t.Errorf("campaignSelectClause is missing %q — TARGET_SPEND campaigns will drift on import (issue #39)", want)
	}
}

// TestCreateCampaign_TargetSpendWithCeiling locks in the write
// path for issue #39: the cpc_bid_ceiling_micros must reach the
// TargetSpend struct on the wire, not just sit in the input
// struct unused.
func TestCreateCampaign_TargetSpendWithCeiling(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	v := int64(8_000_000)
	in := CampaignInput{
		CustomerID:             "123",
		Name:                   "tf-target-spend",
		AdvertisingChannelType: "SEARCH",
		CampaignBudget:         "customers/123/campaignBudgets/1",
		BiddingStrategyType:    "TARGET_SPEND",
		CpcBidCeilingMicros:    &v,
	}
	if _, err := c.CreateCampaign(context.Background(), in); err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}
	create := ts.campaignOps[0].GetCreate()
	ts2, ok := create.GetCampaignBiddingStrategy().(*resources.Campaign_TargetSpend)
	if !ok {
		t.Fatalf("strategy = %T, want TargetSpend", create.GetCampaignBiddingStrategy())
	}
	if got := ts2.TargetSpend.GetCpcBidCeilingMicros(); got != 8_000_000 {
		t.Errorf("CpcBidCeilingMicros on wire = %d, want 8_000_000 (issue #39)", got)
	}
}

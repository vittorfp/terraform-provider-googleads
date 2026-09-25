package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAdGroupCriterionResource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	base := fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-kw-budget"
  amount_micros = 1000000
}
resource "googleads_campaign" "c" {
  customer_id              = %q
  name                     = "tf-acc-kw-campaign"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.b.id
  status                   = "PAUSED"
}
resource "googleads_ad_group" "g" {
  customer_id    = %q
  campaign_id    = googleads_campaign.c.id
  name           = "tf-acc-kw-ag"
  cpc_bid_micros = 250000
}`, cid, cid, cid)

	cfg := base + fmt.Sprintf(`
resource "googleads_ad_group_criterion" "k" {
  customer_id  = %q
  ad_group_id  = googleads_ad_group.g.id
  keyword_text = "blue suede shoes"
  match_type   = "EXACT"
  status       = "ENABLED"
}`, cid)

	paused := base + fmt.Sprintf(`
resource "googleads_ad_group_criterion" "k" {
  customer_id  = %q
  ad_group_id  = googleads_ad_group.g.id
  keyword_text = "blue suede shoes"
  match_type   = "EXACT"
  status       = "PAUSED"
}`, cid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttr("googleads_ad_group_criterion.k", "status", "ENABLED")},
			{Config: paused, Check: resource.TestCheckResourceAttr("googleads_ad_group_criterion.k", "status", "PAUSED")},
			{ResourceName: "googleads_ad_group_criterion.k", ImportState: true, ImportStateVerify: true},
		},
	})
}

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAdGroupAdResource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	cfg := fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-rsa-budget"
  amount_micros = 1000000
}
resource "googleads_campaign" "c" {
  customer_id              = %q
  name                     = "tf-acc-rsa-campaign"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.b.id
  status                   = "PAUSED"
}
resource "googleads_ad_group" "g" {
  customer_id    = %q
  campaign_id    = googleads_campaign.c.id
  name           = "tf-acc-rsa-ag"
  cpc_bid_micros = 250000
}
resource "googleads_ad_group_ad" "a" {
  customer_id  = %q
  ad_group_id  = googleads_ad_group.g.id
  final_urls   = ["https://example.com"]
  headlines    = ["A nice headline", "Another headline", "Yet another headline"]
  descriptions = ["First description here.", "Second description here."]
  status       = "PAUSED"
}`, cid, cid, cid, cid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttrSet("googleads_ad_group_ad.a", "id")},
			{ResourceName: "googleads_ad_group_ad.a", ImportState: true, ImportStateVerify: true},
		},
	})
}

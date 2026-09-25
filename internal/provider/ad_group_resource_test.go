package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAdGroupResource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	base := fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-ag-budget"
  amount_micros = 1000000
}
resource "googleads_campaign" "c" {
  customer_id              = %q
  name                     = "tf-acc-ag-campaign"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.b.id
  status                   = "PAUSED"
}`, cid, cid)

	cfg := base + fmt.Sprintf(`
resource "googleads_ad_group" "g" {
  customer_id    = %q
  campaign_id    = googleads_campaign.c.id
  name           = "tf-acc-ag"
  cpc_bid_micros = 250000
}`, cid)

	updated := base + fmt.Sprintf(`
resource "googleads_ad_group" "g" {
  customer_id    = %q
  campaign_id    = googleads_campaign.c.id
  name           = "tf-acc-ag-renamed"
  cpc_bid_micros = 400000
}`, cid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttr("googleads_ad_group.g", "name", "tf-acc-ag")},
			{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("googleads_ad_group.g", "name", "tf-acc-ag-renamed"),
				resource.TestCheckResourceAttr("googleads_ad_group.g", "cpc_bid_micros", "400000"),
			)},
			{ResourceName: "googleads_ad_group.g", ImportState: true, ImportStateVerify: true},
		},
	})
}

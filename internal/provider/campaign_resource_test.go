package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCampaignResource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	cfgEnabled := fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-campaign-budget"
  amount_micros = 1000000
}
resource "googleads_campaign" "c" {
  customer_id              = %q
  name                     = "tf-acc-campaign"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.b.id
  status                   = "PAUSED"
}`, cid, cid)

	cfgRenamed := fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-campaign-budget"
  amount_micros = 1000000
}
resource "googleads_campaign" "c" {
  customer_id              = %q
  name                     = "tf-acc-campaign-2"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.b.id
  status                   = "PAUSED"
}`, cid, cid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfgEnabled, Check: resource.TestCheckResourceAttrSet("googleads_campaign.c", "id")},
			{Config: cfgRenamed, Check: resource.TestCheckResourceAttr("googleads_campaign.c", "name", "tf-acc-campaign-2")},
			{ResourceName: "googleads_campaign.c", ImportState: true, ImportStateVerify: true},
		},
	})
}

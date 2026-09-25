package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCampaignBudgetResource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-budget"
  amount_micros = 1000000
}`, cid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("googleads_campaign_budget.b", "id"),
					resource.TestCheckResourceAttr("googleads_campaign_budget.b", "name", "tf-acc-budget"),
					resource.TestCheckResourceAttr("googleads_campaign_budget.b", "amount_micros", "1000000"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "googleads_campaign_budget" "b" {
  customer_id   = %q
  name          = "tf-acc-budget-renamed"
  amount_micros = 2000000
}`, cid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("googleads_campaign_budget.b", "name", "tf-acc-budget-renamed"),
					resource.TestCheckResourceAttr("googleads_campaign_budget.b", "amount_micros", "2000000"),
				),
			},
			{
				ResourceName:      "googleads_campaign_budget.b",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

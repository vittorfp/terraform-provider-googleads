package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomerDataSource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "googleads_customer" "self" {
  customer_id = %q
}`, cid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.googleads_customer.self", "currency_code"),
					resource.TestCheckResourceAttrSet("data.googleads_customer.self", "time_zone"),
					resource.TestCheckResourceAttr("data.googleads_customer.self", "resource_name", fmt.Sprintf("customers/%s", cid)),
				),
			},
		},
	})
}

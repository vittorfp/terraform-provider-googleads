package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTextAssetResource(t *testing.T) {
	requireAccEnv(t)
	cid := testCustomerID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "googleads_text_asset" "a" {
  customer_id = %q
  name        = "tf-acc-headline"
  text        = "Discover great products"
}`, cid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("googleads_text_asset.a", "id"),
					resource.TestCheckResourceAttr("googleads_text_asset.a", "text", "Discover great products"),
				),
			},
			{
				// Rename — same text (immutable), new name (mutable).
				Config: fmt.Sprintf(`
resource "googleads_text_asset" "a" {
  customer_id = %q
  name        = "tf-acc-headline-renamed"
  text        = "Discover great products"
}`, cid),
				Check: resource.TestCheckResourceAttr("googleads_text_asset.a", "name", "tf-acc-headline-renamed"),
			},
			{
				ResourceName:      "googleads_text_asset.a",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

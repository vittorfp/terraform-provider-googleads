resource "googleads_customer_asset" "callout" {
  customer_id = "1234567890"
  asset_id    = googleads_callout_asset.free_shipping.id
  field_type  = "CALLOUT"
}

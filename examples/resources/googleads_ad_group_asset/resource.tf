resource "googleads_ad_group_asset" "callout" {
  customer_id = "1234567890"
  ad_group_id = googleads_ad_group.brand.id
  asset_id    = googleads_callout_asset.free_shipping.id
  field_type  = "CALLOUT"
}

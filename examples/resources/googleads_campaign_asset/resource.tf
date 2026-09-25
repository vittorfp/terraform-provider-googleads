resource "googleads_campaign_asset" "callout" {
  customer_id = "1234567890"
  campaign_id = googleads_campaign.search.id
  asset_id    = googleads_callout_asset.free_shipping.id
  field_type  = "CALLOUT"
}

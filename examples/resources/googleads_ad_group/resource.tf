resource "googleads_ad_group" "main" {
  customer_id    = "1234567890"
  campaign_id    = googleads_campaign.search.id
  name           = "tf-ad-group"
  status         = "ENABLED"
  cpc_bid_micros = 500000
}

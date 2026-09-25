resource "googleads_sitelink_asset" "support" {
  customer_id  = "1234567890"
  name         = "Support sitelink"
  link_text    = "24/7 Support"
  description1 = "Chat with us anytime"
  description2 = "Response under 5 minutes"

  final_urls = ["https://example.com/support"]
}

resource "googleads_campaign_asset" "attach_support" {
  customer_id = "1234567890"
  campaign_id = googleads_campaign.search.id
  asset_id    = googleads_sitelink_asset.support.id
  field_type  = "SITELINK"
}

resource "googleads_asset_group" "main" {
  customer_id = "1234567890"
  campaign_id = googleads_campaign.pmax.id # a PERFORMANCE_MAX campaign
  name        = "tf-asset-group"
  status      = "PAUSED"

  final_urls = ["https://example.com"]
  path1      = "deals"
}

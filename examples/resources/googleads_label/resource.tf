resource "googleads_label" "high_priority" {
  customer_id      = "1234567890"
  name             = "High Priority"
  background_color = "#FF0000"
  description      = "Campaigns the strategy team is actively iterating on."
}

resource "googleads_campaign_label" "main" {
  customer_id = "1234567890"
  parent      = googleads_campaign.search.id
  label_id    = googleads_label.high_priority.id
}

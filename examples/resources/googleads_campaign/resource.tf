resource "googleads_campaign" "search" {
  customer_id              = "1234567890"
  name                     = "tf-search"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.daily.id
  status                   = "PAUSED"
  bidding_strategy_type    = "MANUAL_CPC"

  # Required by the Google Ads API to create campaigns in affected accounts.
  contains_eu_political_advertising = "DOES_NOT_CONTAIN_EU_POLITICAL_ADVERTISING"

  # Lock the campaign to Google Search proper — no Search Partners, no Display
  # Network. Pinning these in Terraform keeps the campaign from drifting if
  # Google ever changes its create-time defaults.
  network_settings = {
    target_google_search   = true
    target_search_network  = false
    target_content_network = false
  }

  # Match users only when they are physically in the targeted geos (and
  # exclude only physically-present users from negatives). The default
  # PRESENCE_OR_INTEREST also counts "showed interest in," which is broader
  # than most measurement frameworks want.
  geo_target_type_setting = {
    positive_geo_target_type = "PRESENCE_OR_INTEREST"
    negative_geo_target_type = "PRESENCE"
  }
}

# A Performance Max campaign with Target ROAS smart bidding.
resource "googleads_campaign" "pmax" {
  customer_id              = "1234567890"
  name                     = "tf-pmax"
  advertising_channel_type = "PERFORMANCE_MAX"
  campaign_budget_id       = googleads_campaign_budget.daily.id
  status                   = "PAUSED"

  bidding_strategy_type  = "TARGET_ROAS"
  target_roas            = 3.5
  cpc_bid_ceiling_micros = 5000000
}

# Bid 1.5x more for the 25-34 age range.
resource "googleads_ad_group_audience_criterion" "boost_25_34" {
  customer_id    = "1234567890"
  ad_group_id    = googleads_ad_group.main.id
  age_range_type = "AGE_RANGE_25_34"
  bid_modifier   = 1.5
}

# Target a remarketing list (created in the Ads UI for now).
resource "googleads_ad_group_audience_criterion" "previous_visitors" {
  customer_id  = "1234567890"
  ad_group_id  = googleads_ad_group.main.id
  user_list_id = "customers/1234567890/userLists/12345678"
}

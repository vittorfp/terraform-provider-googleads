resource "googleads_ad_group_criterion" "shoes" {
  customer_id  = "1234567890"
  ad_group_id  = googleads_ad_group.main.id
  keyword_text = "running shoes"
  match_type   = "EXACT"
  status       = "ENABLED"
}

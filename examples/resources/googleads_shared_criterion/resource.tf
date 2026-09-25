resource "googleads_shared_criterion" "free" {
  customer_id   = "1234567890"
  shared_set_id = googleads_shared_set.brand_negatives.id
  keyword_text  = "free"
  match_type    = "BROAD"
}

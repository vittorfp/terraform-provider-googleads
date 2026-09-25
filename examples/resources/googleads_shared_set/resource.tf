resource "googleads_shared_set" "brand_negatives" {
  customer_id = "1234567890"
  name        = "Brand exclusions"
  type        = "NEGATIVE_KEYWORDS"
}

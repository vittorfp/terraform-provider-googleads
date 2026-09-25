# Exclude a specific placement account-wide.
resource "googleads_customer_negative_criterion" "block_site" {
  customer_id   = "1234567890"
  placement_url = "http://www.badsite.com"
}

# Attach a shared negative-keyword list at the customer level.
resource "googleads_customer_negative_criterion" "brand_negatives" {
  customer_id              = "1234567890"
  negative_keyword_list_id = googleads_shared_set.brand_negatives.id
}

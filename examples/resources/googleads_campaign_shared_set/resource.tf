resource "googleads_campaign_shared_set" "attach_brand_negatives" {
  customer_id   = "1234567890"
  campaign_id   = googleads_campaign.search.id
  shared_set_id = googleads_shared_set.brand_negatives.id
}

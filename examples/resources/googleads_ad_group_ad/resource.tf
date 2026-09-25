resource "googleads_ad_group_ad" "rsa" {
  customer_id = "1234567890"
  ad_group_id = googleads_ad_group.main.id
  status      = "ENABLED"

  final_urls = ["https://example.com"]

  headlines = [
    "Discover great products",
    "Shop the latest deals",
    "Free shipping over $50",
  ]

  descriptions = [
    "Curated picks updated daily.",
    "Loved by 10,000+ customers.",
  ]

  path1 = "deals"
  path2 = "today"
}

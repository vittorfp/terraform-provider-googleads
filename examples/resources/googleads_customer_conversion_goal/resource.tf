# Make WEBSITE PURCHASE conversions biddable — i.e. Smart Bidding
# strategies (TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS) on this
# customer will optimize toward them. Required to make a Performance
# Max campaign actually serve.
resource "googleads_customer_conversion_goal" "website_purchase" {
  customer_id = "1234567890"
  category    = "PURCHASE"
  origin      = "WEBSITE"
  biddable    = true
}

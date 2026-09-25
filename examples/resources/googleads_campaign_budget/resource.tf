resource "googleads_campaign_budget" "daily" {
  customer_id   = "1234567890"
  name          = "tf-daily-budget"
  amount_micros = 5000000 # 5 units of account currency per day
}

# More readable variant: declare the daily cap in currency units and
# convert. 1,000,000 micros == 1 unit (USD, BRL, EUR — whichever
# currency the customer was created in).
locals {
  currency_to_micros = 1000000
}

resource "googleads_campaign_budget" "daily_readable" {
  customer_id   = "1234567890"
  name          = "tf-daily-budget-readable"
  amount_micros = 50 * local.currency_to_micros # = 50,000,000 micros
}

# A standard website purchase conversion. Pair with a smart-bidding strategy
# (TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS, MAXIMIZE_CONVERSION_VALUE)
# on a googleads_campaign for the conversion to influence bidding.
resource "googleads_conversion_action" "purchase" {
  customer_id   = "1234567890"
  name          = "Website Purchase"
  type          = "WEBPAGE"
  category      = "PURCHASE"
  counting_type = "ONE_PER_CLICK"

  click_through_lookback_window_days = 30
  view_through_lookback_window_days  = 1

  primary_for_goal              = true
  include_in_conversions_metric = true

  # Optional default value for events reported without their own value.
  default_value         = 50.00
  default_currency_code = "USD"
}

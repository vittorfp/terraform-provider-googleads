# Target the United States.
resource "googleads_campaign_criterion" "us" {
  customer_id = "1234567890"
  campaign_id = googleads_campaign.search.id
  location_id = "geoTargetConstants/2840"
}

# Bid 25% more on mobile.
resource "googleads_campaign_criterion" "mobile_boost" {
  customer_id  = "1234567890"
  campaign_id  = googleads_campaign.search.id
  device_type  = "MOBILE"
  bid_modifier = 1.25
}

# Run weekdays 9am–6pm.
resource "googleads_campaign_criterion" "weekday_business_hours" {
  customer_id              = "1234567890"
  campaign_id              = googleads_campaign.search.id
  ad_schedule_day_of_week  = "MONDAY"
  ad_schedule_start_hour   = 9
  ad_schedule_end_hour     = 18
  ad_schedule_start_minute = "ZERO"
  ad_schedule_end_minute   = "ZERO"
}

# Target a 7km radius around downtown Belo Horizonte.
resource "googleads_campaign_criterion" "bh_centro" {
  customer_id          = "1234567890"
  campaign_id          = googleads_campaign.search.id
  latitude             = -19.911
  longitude            = -43.985
  radius               = 7.0
  radius_units         = "KILOMETERS"
  address_country_code = "BR"
  address_city_name    = "Belo Horizonte"
}

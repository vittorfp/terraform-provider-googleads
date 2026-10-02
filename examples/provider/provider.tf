terraform {
  required_providers {
    googleads = {
      source  = "vittorfp/googleads"
      version = "~> 0.6"
    }
  }
}

# All credentials may instead be left blank and supplied via env vars:
#   GOOGLE_ADS_LOGIN_CUSTOMER_ID  # only for manager-account access
#   GOOGLE_ADS_CLIENT_ID
#   GOOGLE_ADS_CLIENT_SECRET
#   GOOGLE_ADS_REFRESH_TOKEN
provider "googleads" {
  login_customer_id = var.login_customer_id
  client_id         = var.client_id
  client_secret     = var.client_secret
  refresh_token     = var.refresh_token
}

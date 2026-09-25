resource "googleads_asset_group_asset" "headline_1" {
  customer_id    = "1234567890"
  asset_group_id = googleads_asset_group.main.id
  asset_id       = googleads_text_asset.headline.id
  field_type     = "HEADLINE"
}

resource "googleads_asset_group_asset" "logo" {
  customer_id    = "1234567890"
  asset_group_id = googleads_asset_group.main.id
  asset_id       = googleads_image_asset.logo.id
  field_type     = "LOGO"
}

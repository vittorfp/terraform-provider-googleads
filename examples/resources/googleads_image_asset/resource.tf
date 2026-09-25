resource "googleads_image_asset" "logo" {
  customer_id = "1234567890"
  name        = "tf-logo"
  path        = "${path.module}/logo.png" # PNG, JPEG, or GIF on disk
}

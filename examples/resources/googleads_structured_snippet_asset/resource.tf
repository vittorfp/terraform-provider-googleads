resource "googleads_structured_snippet_asset" "services" {
  customer_id = "1234567890"
  header      = "Services"
  values      = ["Consulting", "Implementation", "Training", "24/7 support"]
}

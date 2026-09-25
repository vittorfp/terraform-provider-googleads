data "googleads_customer" "self" {
  customer_id = "1234567890"
}

output "currency" {
  value = data.googleads_customer.self.currency_code
}

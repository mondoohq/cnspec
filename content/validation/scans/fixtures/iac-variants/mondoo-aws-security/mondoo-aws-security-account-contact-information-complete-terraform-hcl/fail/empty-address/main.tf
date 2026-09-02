# Non-compliant: the primary contact has an empty address line.
resource "aws_account_primary_contact" "main" {
  full_name      = "Example Corp"
  address_line_1 = ""
  phone_number   = "+49-30-1234567"
  city           = "Berlin"
  country_code   = "DE"
  postal_code    = "10115"
}

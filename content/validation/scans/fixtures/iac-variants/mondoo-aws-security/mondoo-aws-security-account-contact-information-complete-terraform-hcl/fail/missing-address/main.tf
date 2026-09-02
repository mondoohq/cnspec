# Non-compliant: the primary contact omits the address line entirely.
# The sibling fixture covers the same field present but empty; both shapes have
# to fail, and `!= empty` in the check is what covers the pair.
resource "aws_account_primary_contact" "main" {
  full_name    = "Example Corp"
  phone_number = "+49-30-1234567"
  city         = "Berlin"
  country_code = "DE"
  postal_code  = "10115"
}

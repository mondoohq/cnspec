# Non-compliant: alternate contacts are managed, but no SECURITY contact exists.
resource "aws_account_alternate_contact" "billing" {
  alternate_contact_type = "BILLING"
  name                   = "Finance"
  title                  = "Head of Finance"
  email_address          = "billing@example.com"
  phone_number           = "+1-555-0100"
}

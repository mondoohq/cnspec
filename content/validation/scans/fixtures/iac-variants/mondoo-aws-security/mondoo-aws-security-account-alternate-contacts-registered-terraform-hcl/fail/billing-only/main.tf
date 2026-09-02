# Non-compliant: only a billing contact is managed; security and operations are missing.
resource "aws_account_alternate_contact" "billing" {
  alternate_contact_type = "BILLING"
  name                   = "Finance"
  title                  = "Head of Finance"
  email_address          = "billing@example.com"
  phone_number           = "+1-555-0100"
}

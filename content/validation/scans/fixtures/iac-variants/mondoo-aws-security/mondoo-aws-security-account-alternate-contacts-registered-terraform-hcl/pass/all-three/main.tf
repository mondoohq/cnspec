# Compliant: billing, operations, and security alternate contacts are managed.
resource "aws_account_alternate_contact" "security" {
  alternate_contact_type = "SECURITY"
  name                   = "Security Team"
  title                  = "CISO"
  email_address          = "security@example.com"
  phone_number           = "+1-555-0100"
}

resource "aws_account_alternate_contact" "billing" {
  alternate_contact_type = "BILLING"
  name                   = "Finance"
  title                  = "Head of Finance"
  email_address          = "billing@example.com"
  phone_number           = "+1-555-0100"
}

resource "aws_account_alternate_contact" "operations" {
  alternate_contact_type = "OPERATIONS"
  name                   = "Operations"
  title                  = "Head of Operations"
  email_address          = "ops@example.com"
  phone_number           = "+1-555-0100"
}

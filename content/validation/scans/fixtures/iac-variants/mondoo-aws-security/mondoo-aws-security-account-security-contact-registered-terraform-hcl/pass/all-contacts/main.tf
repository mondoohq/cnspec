# Compliant: a SECURITY alternate contact with an email address is managed.
resource "aws_account_alternate_contact" "security" {
  alternate_contact_type = "SECURITY"
  name                   = "Security Team"
  title                  = "CISO"
  email_address          = "security@example.com"
  phone_number           = "+1-555-0100"
}

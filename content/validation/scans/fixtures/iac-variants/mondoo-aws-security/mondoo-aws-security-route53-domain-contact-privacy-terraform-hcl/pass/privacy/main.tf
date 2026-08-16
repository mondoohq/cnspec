# Compliant: privacy protection is enabled for all contact types.
resource "aws_route53domains_registered_domain" "pass_example" {
  domain_name       = "example.com"
  admin_privacy     = true
  registrant_privacy = true
  tech_privacy      = true
}

# Non-compliant: registrant privacy protection is disabled.
resource "aws_route53domains_registered_domain" "fail_example" {
  domain_name        = "example.com"
  admin_privacy      = true
  registrant_privacy = false
  tech_privacy       = true
}

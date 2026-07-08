# Compliant: registered domain has auto renew enabled.
resource "aws_route53domains_registered_domain" "pass_example" {
  domain_name = "example.com"
  auto_renew  = true
}

# Non-compliant: registered domain has auto renew disabled.
resource "aws_route53domains_registered_domain" "fail_example" {
  domain_name = "example.com"
  auto_renew  = false
}

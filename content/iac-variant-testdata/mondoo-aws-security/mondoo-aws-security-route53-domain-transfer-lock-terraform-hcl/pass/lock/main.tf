# Compliant: registered domain has the transfer lock enabled.
resource "aws_route53domains_registered_domain" "pass_example" {
  domain_name   = "example.com"
  transfer_lock = true
}

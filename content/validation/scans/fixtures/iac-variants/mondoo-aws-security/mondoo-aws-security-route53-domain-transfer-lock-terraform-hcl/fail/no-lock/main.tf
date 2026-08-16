# Non-compliant: registered domain has the transfer lock disabled.
resource "aws_route53domains_registered_domain" "fail_example" {
  domain_name   = "example.com"
  transfer_lock = false
}

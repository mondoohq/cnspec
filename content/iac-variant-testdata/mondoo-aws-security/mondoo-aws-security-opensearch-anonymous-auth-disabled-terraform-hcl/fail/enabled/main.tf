resource "aws_opensearch_domain" "example" {
  domain_name = "example"

  advanced_security_options {
    enabled                = true
    anonymous_auth_enabled = true
  }
}

# Compliant: HTTPS is enforced.
resource "aws_opensearch_domain" "pass_example" {
  domain_name = "example"

  domain_endpoint_options {
    enforce_https = true
  }
}

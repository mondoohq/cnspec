# Non-compliant: HTTPS is not enforced.
resource "aws_opensearch_domain" "fail_example" {
  domain_name = "example"

  domain_endpoint_options {
    enforce_https = false
  }
}

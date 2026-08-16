# Non-compliant: tls_security_policy is omitted, so it defaults to the weak Policy-Min-TLS-1-0.
resource "aws_opensearch_domain" "fail_no_policy" {
  domain_name = "example"

  domain_endpoint_options {
    enforce_https = true
  }
}

# Non-compliant: endpoint allows the weaker TLS 1.0 policy.
resource "aws_opensearch_domain" "fail_example" {
  domain_name = "example"

  domain_endpoint_options {
    tls_security_policy = "Policy-Min-TLS-1-0-2019-07"
  }
}

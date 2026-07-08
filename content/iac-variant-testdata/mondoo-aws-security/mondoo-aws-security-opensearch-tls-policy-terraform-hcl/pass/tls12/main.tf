# Compliant: endpoint enforces a minimum TLS 1.2 policy.
resource "aws_opensearch_domain" "pass_example" {
  domain_name = "example"

  domain_endpoint_options {
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }
}

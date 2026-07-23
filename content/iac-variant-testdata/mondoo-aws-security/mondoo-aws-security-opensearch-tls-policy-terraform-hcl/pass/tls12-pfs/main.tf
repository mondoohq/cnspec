# Compliant: endpoint enforces a minimum TLS 1.2 policy with perfect forward secrecy.
resource "aws_opensearch_domain" "pass_pfs" {
  domain_name = "example"

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-PFS-2023-10"
  }
}

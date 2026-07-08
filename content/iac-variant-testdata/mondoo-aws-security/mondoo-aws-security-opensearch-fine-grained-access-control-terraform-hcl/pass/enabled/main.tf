# Compliant: fine-grained access control is enabled.
resource "aws_opensearch_domain" "pass_example" {
  domain_name = "example"

  advanced_security_options {
    enabled = true
  }
}

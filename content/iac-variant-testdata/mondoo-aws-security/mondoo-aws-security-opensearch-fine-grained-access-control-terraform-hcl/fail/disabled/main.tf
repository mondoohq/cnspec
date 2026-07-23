# Non-compliant: fine-grained access control is disabled.
resource "aws_opensearch_domain" "fail_example" {
  domain_name = "example"

  advanced_security_options {
    enabled = false
  }
}

# Compliant: encryption at rest is enabled.
resource "aws_opensearch_domain" "pass_example" {
  domain_name = "example"

  encrypt_at_rest {
    enabled = true
  }
}

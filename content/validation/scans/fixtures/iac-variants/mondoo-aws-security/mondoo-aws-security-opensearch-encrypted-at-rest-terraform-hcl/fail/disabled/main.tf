# Non-compliant: encryption at rest is disabled.
resource "aws_opensearch_domain" "fail_example" {
  domain_name = "example"

  encrypt_at_rest {
    enabled = false
  }
}

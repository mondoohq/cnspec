resource "aws_opensearch_domain" "example" {
  domain_name = "example"

  encrypt_at_rest {
    enabled = true
  }
}

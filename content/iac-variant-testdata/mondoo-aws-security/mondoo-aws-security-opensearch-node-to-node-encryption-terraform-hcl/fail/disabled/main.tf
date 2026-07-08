# Non-compliant: node-to-node encryption is disabled.
resource "aws_opensearch_domain" "fail_example" {
  domain_name = "example"

  node_to_node_encryption {
    enabled = false
  }
}

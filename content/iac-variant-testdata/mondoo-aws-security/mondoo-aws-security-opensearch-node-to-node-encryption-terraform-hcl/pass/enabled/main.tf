# Compliant: node-to-node encryption is enabled.
resource "aws_opensearch_domain" "pass_example" {
  domain_name = "example"

  node_to_node_encryption {
    enabled = true
  }
}

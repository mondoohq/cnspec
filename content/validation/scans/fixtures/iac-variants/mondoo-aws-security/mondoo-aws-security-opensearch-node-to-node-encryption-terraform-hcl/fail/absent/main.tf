# Non-compliant: no node_to_node_encryption block, so it defaults to disabled.
resource "aws_opensearch_domain" "fail_absent" {
  domain_name    = "example"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type = "t3.small.search"
  }
}

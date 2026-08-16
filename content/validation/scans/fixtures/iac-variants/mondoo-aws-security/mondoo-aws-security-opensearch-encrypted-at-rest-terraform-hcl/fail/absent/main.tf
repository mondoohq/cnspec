# Non-compliant: no encrypt_at_rest block at all, so encryption is left off.
resource "aws_opensearch_domain" "fail_absent" {
  domain_name    = "example"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type  = "t3.small.search"
    instance_count = 1
  }
}

# Non-compliant: no domain_endpoint_options block, so HTTPS is not enforced.
resource "aws_opensearch_domain" "fail_absent" {
  domain_name    = "example"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type = "t3.small.search"
  }
}

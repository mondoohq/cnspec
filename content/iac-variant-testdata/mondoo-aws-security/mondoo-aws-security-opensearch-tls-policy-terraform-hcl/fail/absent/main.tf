# Non-compliant: no domain_endpoint_options block, so the endpoint uses the default weak TLS policy.
resource "aws_opensearch_domain" "fail_absent" {
  domain_name    = "example"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type = "t3.small.search"
  }
}

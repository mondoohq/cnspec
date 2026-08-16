# Non-compliant: no advanced_security_options block, so fine-grained access control is off.
resource "aws_opensearch_domain" "fail_absent" {
  domain_name    = "example"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type = "t3.small.search"
  }
}

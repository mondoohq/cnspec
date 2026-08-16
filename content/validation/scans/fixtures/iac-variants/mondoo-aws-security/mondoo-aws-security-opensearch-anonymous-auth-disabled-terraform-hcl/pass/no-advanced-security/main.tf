resource "aws_opensearch_domain" "example" {
  domain_name    = "example"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type = "r6g.large.search"
  }
}

# Non-compliant: domain has no vpc_options block (public endpoint).
resource "aws_opensearch_domain" "fail_example" {
  domain_name = "example"

  cluster_config {
    instance_type = "t3.small.search"
  }
}

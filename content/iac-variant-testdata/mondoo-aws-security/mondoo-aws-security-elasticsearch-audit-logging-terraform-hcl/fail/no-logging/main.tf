# Non-compliant: no log publishing configured at all.
resource "aws_elasticsearch_domain" "fail_example" {
  domain_name           = "fail-example"
  elasticsearch_version = "7.10"

  cluster_config {
    instance_type = "t3.small.elasticsearch"
  }
}

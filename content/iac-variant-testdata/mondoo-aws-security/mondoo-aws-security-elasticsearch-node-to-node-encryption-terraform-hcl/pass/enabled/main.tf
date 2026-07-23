resource "aws_elasticsearch_domain" "pass" {
  domain_name           = "example"
  elasticsearch_version = "7.10"

  node_to_node_encryption {
    enabled = true
  }
}

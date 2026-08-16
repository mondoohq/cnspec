# Two domains; the second disables node-to-node encryption, so .all() must fail.
resource "aws_elasticsearch_domain" "compliant" {
  domain_name           = "compliant"
  elasticsearch_version = "7.10"

  node_to_node_encryption {
    enabled = true
  }
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  node_to_node_encryption {
    enabled = false
  }
}

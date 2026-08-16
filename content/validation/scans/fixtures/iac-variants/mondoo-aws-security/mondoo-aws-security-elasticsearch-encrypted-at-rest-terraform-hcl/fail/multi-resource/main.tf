# Two domains; the second disables encryption at rest, so .all() must fail.
resource "aws_elasticsearch_domain" "compliant" {
  domain_name           = "compliant"
  elasticsearch_version = "7.10"

  encrypt_at_rest {
    enabled = true
  }
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  encrypt_at_rest {
    enabled = false
  }
}

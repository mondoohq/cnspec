# Two domains; the second does not enforce HTTPS, so .all() must fail.
resource "aws_elasticsearch_domain" "compliant" {
  domain_name           = "compliant"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https = true
  }
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https = false
  }
}

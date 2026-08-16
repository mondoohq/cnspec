# Two domains; the second uses a legacy TLS 1.0 policy, so .all() must fail.
resource "aws_elasticsearch_domain" "compliant" {
  domain_name           = "compliant"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-0-2019-07"
  }
}

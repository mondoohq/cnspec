# Non-compliant: TLS policy chosen by a ternary whose active branch is the legacy policy.
variable "legacy" {
  type    = bool
  default = true
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = var.legacy ? "Policy-Min-TLS-1-0-2019-07" : "Policy-Min-TLS-1-2-2019-07"
  }
}

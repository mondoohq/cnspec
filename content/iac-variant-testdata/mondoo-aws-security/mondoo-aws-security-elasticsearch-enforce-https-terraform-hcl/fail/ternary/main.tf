# Non-compliant: enforce_https toggled by a ternary whose active branch is false.
variable "public" {
  type    = bool
  default = true
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https = var.public ? false : true
  }
}

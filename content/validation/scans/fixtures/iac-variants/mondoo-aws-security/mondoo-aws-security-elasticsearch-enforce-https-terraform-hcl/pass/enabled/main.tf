resource "aws_elasticsearch_domain" "pass" {
  domain_name           = "example"
  elasticsearch_version = "7.10"

  domain_endpoint_options {
    enforce_https = true
  }
}

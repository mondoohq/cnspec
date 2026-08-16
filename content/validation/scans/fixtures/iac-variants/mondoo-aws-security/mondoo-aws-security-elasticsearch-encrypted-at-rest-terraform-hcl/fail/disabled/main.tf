resource "aws_elasticsearch_domain" "fail" {
  domain_name           = "example"
  elasticsearch_version = "7.10"

  encrypt_at_rest {
    enabled = false
  }
}

resource "aws_elasticsearch_domain" "pass" {
  domain_name           = "example"
  elasticsearch_version = "7.10"

  encrypt_at_rest {
    enabled = true
  }
}

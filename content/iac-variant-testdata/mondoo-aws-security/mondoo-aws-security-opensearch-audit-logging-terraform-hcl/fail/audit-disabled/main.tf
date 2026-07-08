resource "aws_opensearch_domain" "example" {
  domain_name = "example"

  log_publishing_options {
    log_type                 = "AUDIT_LOGS"
    cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:123456789012:log-group:example"
    enabled                  = false
  }
}

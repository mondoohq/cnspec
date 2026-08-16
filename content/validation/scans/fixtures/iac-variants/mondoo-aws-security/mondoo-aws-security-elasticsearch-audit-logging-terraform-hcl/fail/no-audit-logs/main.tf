# Non-compliant: only slow logs published, no audit logs.
resource "aws_elasticsearch_domain" "fail_example" {
  domain_name           = "fail-example"
  elasticsearch_version = "7.10"

  log_publishing_options {
    log_type                 = "INDEX_SLOW_LOGS"
    cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:111122223333:log-group:slow"
  }
}

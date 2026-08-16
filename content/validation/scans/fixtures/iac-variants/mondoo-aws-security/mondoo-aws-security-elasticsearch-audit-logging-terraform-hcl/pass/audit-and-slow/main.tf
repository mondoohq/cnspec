# Compliant: audit logs published alongside slow logs (a realistic domain
# publishes several log types). Audit logging IS enabled here.
resource "aws_elasticsearch_domain" "pass_example" {
  domain_name           = "pass-example"
  elasticsearch_version = "7.10"

  log_publishing_options {
    log_type                 = "INDEX_SLOW_LOGS"
    cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:111122223333:log-group:slow"
  }

  log_publishing_options {
    log_type                 = "AUDIT_LOGS"
    cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:111122223333:log-group:audit"
  }
}

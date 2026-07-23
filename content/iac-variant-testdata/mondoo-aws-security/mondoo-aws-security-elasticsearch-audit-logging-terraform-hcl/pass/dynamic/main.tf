# Compliant: audit logs published via a dynamic "log_publishing_options" block.
variable "log_types" {
  type    = set(string)
  default = ["AUDIT_LOGS", "INDEX_SLOW_LOGS"]
}

resource "aws_elasticsearch_domain" "pass_dynamic" {
  domain_name           = "pass-dynamic"
  elasticsearch_version = "7.10"

  dynamic "log_publishing_options" {
    for_each = var.log_types
    content {
      log_type                 = log_publishing_options.value
      cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:111122223333:log-group:${log_publishing_options.value}"
    }
  }
}

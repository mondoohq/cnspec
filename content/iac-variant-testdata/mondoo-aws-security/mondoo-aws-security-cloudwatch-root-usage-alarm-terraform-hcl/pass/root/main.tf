# Compliant: metric filter matches root account usage.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "root-usage"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ $.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }"

  metric_transformation {
    name      = "RootUsageEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

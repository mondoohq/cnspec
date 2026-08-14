resource "aws_networkfirewall_logging_configuration" "perimeter" {
  firewall_arn = aws_networkfirewall_firewall.perimeter.arn

  logging_configuration {
    log_destination_config {
      log_destination_type = "S3"
      log_type             = "FLOW"

      log_destination = {
        bucketName = aws_s3_bucket.firewall_logs.bucket
        prefix     = "flow/"
      }
    }

    log_destination_config {
      log_destination_type = "CloudWatchLogs"
      log_type             = "ALERT"

      log_destination = {
        logGroup = aws_cloudwatch_log_group.firewall_alerts.name
      }
    }
  }
}

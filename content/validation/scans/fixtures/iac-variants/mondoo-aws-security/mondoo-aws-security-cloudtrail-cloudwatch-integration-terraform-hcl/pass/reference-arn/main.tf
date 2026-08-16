# Compliant: multi-region trail forwards to a CloudWatch Logs group referenced
# by resource attribute rather than a hard-coded ARN.
resource "aws_cloudwatch_log_group" "trail" {
  name              = "cloudtrail-logs"
  retention_in_days = 90
}

resource "aws_cloudtrail" "pass_example" {
  name                       = "example"
  s3_bucket_name             = "example-bucket"
  is_multi_region_trail      = true
  cloud_watch_logs_group_arn = "${aws_cloudwatch_log_group.trail.arn}:*"
  cloud_watch_logs_role_arn  = aws_iam_role.trail.arn
}

resource "aws_iam_role" "trail" {
  name               = "cloudtrail-to-cloudwatch"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "cloudtrail.amazonaws.com" }
        Action    = "sts:AssumeRole"
      }
    ]
  })
}

# Compliant: log group encrypted with a customer-managed KMS key created in the same config.
resource "aws_kms_key" "logs" {
  description             = "KMS key for CloudWatch log group encryption"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_cloudwatch_log_group" "pass_example" {
  name       = "example-log-group"
  kms_key_id = aws_kms_key.logs.arn
}

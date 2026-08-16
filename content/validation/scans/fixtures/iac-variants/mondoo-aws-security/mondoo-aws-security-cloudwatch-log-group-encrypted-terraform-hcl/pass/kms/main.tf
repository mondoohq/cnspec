# Compliant: log group is encrypted with a KMS key.
resource "aws_cloudwatch_log_group" "pass_example" {
  name       = "example-log-group"
  kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-1234-1234-1234-1234567890ab"
}

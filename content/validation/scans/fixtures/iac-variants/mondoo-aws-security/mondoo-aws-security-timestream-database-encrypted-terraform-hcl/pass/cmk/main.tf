# Compliant: Timestream database encrypted with a customer-managed KMS key.
resource "aws_timestreamwrite_database" "enc" {
  database_name = "metrics"
  kms_key_id    = aws_kms_key.example.arn
}

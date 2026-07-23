# Compliant: SNS topic is encrypted with a customer-managed KMS key.
resource "aws_sns_topic" "example" {
  name              = "example-topic"
  kms_master_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
}

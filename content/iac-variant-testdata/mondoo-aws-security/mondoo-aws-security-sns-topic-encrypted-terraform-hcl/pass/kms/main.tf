# Compliant: SNS topic sets a KMS master key, enabling server-side encryption.
resource "aws_sns_topic" "example" {
  name              = "example-topic"
  kms_master_key_id = "alias/aws/sns"
}

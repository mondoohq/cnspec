# Non-compliant: SNS topic uses the AWS-managed key (alias/aws/sns) instead of a CMK.
resource "aws_sns_topic" "example" {
  name              = "example-topic"
  kms_master_key_id = "alias/aws/sns"
}

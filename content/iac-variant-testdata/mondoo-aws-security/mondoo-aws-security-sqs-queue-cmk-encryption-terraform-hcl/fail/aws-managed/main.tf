# Non-compliant: queue uses the AWS-managed key alias, not a customer-managed key.
resource "aws_sqs_queue" "fail_example" {
  name              = "example-queue"
  kms_master_key_id = "alias/aws/sqs"
}

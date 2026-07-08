# Compliant: queue is encrypted with a KMS key.
resource "aws_sqs_queue" "pass_example" {
  name              = "example-queue"
  kms_master_key_id = "alias/aws/sqs"
}

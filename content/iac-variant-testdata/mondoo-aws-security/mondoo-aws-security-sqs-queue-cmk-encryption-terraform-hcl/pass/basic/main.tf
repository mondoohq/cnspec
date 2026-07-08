# Compliant: queue is encrypted with a customer-managed KMS key.
resource "aws_sqs_queue" "pass_example" {
  name              = "example-queue"
  kms_master_key_id = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
}

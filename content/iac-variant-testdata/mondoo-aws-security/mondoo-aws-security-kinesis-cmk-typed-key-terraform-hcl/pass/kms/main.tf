# Compliant: KMS encryption with a customer managed key.
resource "aws_kinesis_stream" "pass_example" {
  name             = "pass-example"
  shard_count      = 1
  encryption_type  = "KMS"
  kms_key_id       = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
}

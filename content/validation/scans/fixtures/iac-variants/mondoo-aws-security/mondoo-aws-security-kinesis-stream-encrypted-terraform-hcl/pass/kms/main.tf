# Compliant: KMS encryption enabled.
resource "aws_kinesis_stream" "pass_example" {
  name             = "pass-example"
  shard_count      = 1
  encryption_type  = "KMS"
  kms_key_id       = "alias/aws/kinesis"
}

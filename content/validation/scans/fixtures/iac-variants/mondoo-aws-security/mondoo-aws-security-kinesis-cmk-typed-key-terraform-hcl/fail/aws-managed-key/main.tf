# Non-compliant: encryption uses the default AWS-managed aws/kinesis service key,
# not a customer-managed KMS key (CMK). The check description says this should be
# flagged, but the mql only asserts kms_key_id != empty.
resource "aws_kinesis_stream" "fail_example" {
  name            = "fail-example"
  shard_count     = 1
  encryption_type = "KMS"
  kms_key_id      = "alias/aws/kinesis"
}

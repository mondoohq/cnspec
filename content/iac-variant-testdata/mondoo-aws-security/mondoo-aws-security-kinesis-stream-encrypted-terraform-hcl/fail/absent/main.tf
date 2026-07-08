# Non-compliant: no encryption_type set, so the stream defaults to unencrypted (NONE).
resource "aws_kinesis_stream" "fail_example" {
  name        = "fail-example"
  shard_count = 1
}

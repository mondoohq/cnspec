# Non-compliant: no encryption.
resource "aws_kinesis_stream" "fail_example" {
  name             = "fail-example"
  shard_count      = 1
  encryption_type  = "NONE"
}

# Non-compliant: encryption disabled.
resource "aws_kinesis_stream" "fail_example" {
  name             = "fail-example"
  shard_count      = 1
  encryption_type  = "NONE"
}

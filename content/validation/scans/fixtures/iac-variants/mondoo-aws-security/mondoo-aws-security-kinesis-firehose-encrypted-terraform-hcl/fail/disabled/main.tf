# Non-compliant: server side encryption disabled.
resource "aws_kinesis_firehose_delivery_stream" "fail_example" {
  name        = "fail-example"
  destination = "extended_s3"

  server_side_encryption {
    enabled = false
  }
}

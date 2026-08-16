# Compliant: server side encryption enabled.
resource "aws_kinesis_firehose_delivery_stream" "pass_example" {
  name        = "pass-example"
  destination = "extended_s3"

  server_side_encryption {
    enabled  = true
    key_type = "AWS_OWNED_CMK"
  }
}

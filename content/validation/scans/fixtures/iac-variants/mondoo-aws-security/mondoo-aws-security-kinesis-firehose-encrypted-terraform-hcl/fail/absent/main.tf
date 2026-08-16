# Non-compliant: no server_side_encryption block, so the delivery stream is unencrypted.
resource "aws_kinesis_firehose_delivery_stream" "fail_example" {
  name        = "fail-example"
  destination = "extended_s3"

  extended_s3_configuration {
    role_arn   = "arn:aws:iam::123456789012:role/firehose"
    bucket_arn = "arn:aws:s3:::example-bucket"
  }
}

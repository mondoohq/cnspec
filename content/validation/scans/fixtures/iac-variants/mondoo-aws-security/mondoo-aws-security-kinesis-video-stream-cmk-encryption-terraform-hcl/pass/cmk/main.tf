# Compliant: customer managed KMS key set.
resource "aws_kinesis_video_stream" "pass_example" {
  name       = "pass-example"
  kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
}

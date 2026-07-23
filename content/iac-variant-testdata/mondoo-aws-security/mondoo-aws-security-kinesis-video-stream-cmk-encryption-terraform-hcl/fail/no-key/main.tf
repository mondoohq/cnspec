# Non-compliant: no customer managed KMS key (defaults to AWS managed key).
resource "aws_kinesis_video_stream" "fail_example" {
  name = "fail-example"
}

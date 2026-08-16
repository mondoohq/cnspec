# Compliant: bucket has access logging configured with a target bucket.
resource "aws_s3_bucket" "pass_example" {
  bucket = "pass-example-bucket"
}

resource "aws_s3_bucket_logging" "pass_example" {
  bucket        = aws_s3_bucket.pass_example.id
  target_bucket = "log-target-bucket"
  target_prefix = "log/"
}

# Non-compliant: bucket has no logging configuration.
resource "aws_s3_bucket" "fail_example" {
  bucket = "fail-example-bucket"
}

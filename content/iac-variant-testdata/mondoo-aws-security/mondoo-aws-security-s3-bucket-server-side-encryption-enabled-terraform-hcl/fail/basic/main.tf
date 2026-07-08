# Non-compliant: encryption configuration resource exists but defines no rule.
resource "aws_s3_bucket" "fail_example" {
  bucket = "fail-example-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "fail_example" {
  bucket = aws_s3_bucket.fail_example.id
}

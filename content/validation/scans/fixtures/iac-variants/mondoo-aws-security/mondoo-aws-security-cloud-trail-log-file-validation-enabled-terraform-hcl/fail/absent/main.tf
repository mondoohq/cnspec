# Non-compliant: log file validation left unset, which defaults to disabled.
resource "aws_cloudtrail" "fail_absent" {
  name           = "example-trail"
  s3_bucket_name = "example-bucket"
}

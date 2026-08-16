# Non-compliant: CloudTrail does not enable log file validation.
resource "aws_cloudtrail" "fail_example" {
  name                          = "example-trail"
  s3_bucket_name                = "example-bucket"
  enable_log_file_validation    = false
}

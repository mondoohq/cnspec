# Compliant: CloudTrail enables log file validation.
resource "aws_cloudtrail" "pass_example" {
  name                          = "example-trail"
  s3_bucket_name                = "example-bucket"
  enable_log_file_validation    = true
}

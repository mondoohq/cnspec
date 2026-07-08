# Non-compliant: CloudTrail has no KMS key configured.
resource "aws_cloudtrail" "fail_example" {
  name           = "example-trail"
  s3_bucket_name = "example-bucket"
}

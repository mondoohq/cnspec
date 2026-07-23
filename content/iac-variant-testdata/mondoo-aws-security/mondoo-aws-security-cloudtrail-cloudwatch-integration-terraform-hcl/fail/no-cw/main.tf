# Non-compliant: multi-region trail lacks CloudWatch Logs integration.
resource "aws_cloudtrail" "fail_example" {
  name                  = "example"
  s3_bucket_name        = "example-bucket"
  is_multi_region_trail = true
}

# enable_logging defaults to true when omitted
resource "aws_cloudtrail" "main" {
  name                  = "main-trail"
  s3_bucket_name        = "my-cloudtrail-bucket"
  is_multi_region_trail = true
}

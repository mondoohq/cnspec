resource "aws_cloudtrail" "main" {
  name                          = "main-trail"
  s3_bucket_name                = "my-cloudtrail-bucket"
  is_multi_region_trail         = true
  include_global_service_events = true
  enable_logging                = true
}

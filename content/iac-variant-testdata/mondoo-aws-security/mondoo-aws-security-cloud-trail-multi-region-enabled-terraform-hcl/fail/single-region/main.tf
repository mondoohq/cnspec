# Non-compliant: trail is limited to a single region.
resource "aws_cloudtrail" "example" {
  name                       = "example"
  s3_bucket_name             = "example-bucket"
  is_multi_region_trail      = false
}

# Non-compliant: is_multi_region_trail omitted, so it defaults to false.
resource "aws_cloudtrail" "example" {
  name           = "example"
  s3_bucket_name = "example-bucket"
}

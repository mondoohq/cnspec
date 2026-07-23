# Compliant: an organization-wide multi-region trail exists alongside a
# supplementary single-region trail.
resource "aws_cloudtrail" "regional" {
  name                  = "regional"
  s3_bucket_name        = "example-bucket"
  is_multi_region_trail = false
}

resource "aws_cloudtrail" "org" {
  name                  = "org-wide"
  s3_bucket_name        = "example-bucket"
  is_multi_region_trail = true
}

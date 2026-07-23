# Non-compliant: S3 origin still uses a legacy origin access identity and has
# no origin access control.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  origin {
    domain_name = "example.s3.amazonaws.com"
    origin_id   = "s3-origin"

    s3_origin_config {
      origin_access_identity = "origin-access-identity/cloudfront/E127EXAMPLE51Z"
    }
  }
}

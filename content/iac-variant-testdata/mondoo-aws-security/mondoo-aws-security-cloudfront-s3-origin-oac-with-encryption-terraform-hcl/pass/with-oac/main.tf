# Compliant: S3 origin uses an origin access control.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  origin {
    domain_name              = "example.s3.amazonaws.com"
    origin_id                = "s3-origin"
    origin_access_control_id = "E2ABCDEF123456"

    s3_origin_config {
      origin_access_identity = ""
    }
  }
}

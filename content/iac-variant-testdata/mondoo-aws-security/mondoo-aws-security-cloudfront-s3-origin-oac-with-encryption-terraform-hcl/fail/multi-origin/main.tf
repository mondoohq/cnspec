# One distribution with two origins; the second S3 origin lacks an OAC, so the
# per-origin .all() must fail (catches any first-origin-only bug).
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  origin {
    domain_name              = "good.s3.amazonaws.com"
    origin_id                = "good-s3"
    origin_access_control_id = "E2ABCOAC12345"
    s3_origin_config {
      origin_access_identity = ""
    }
  }

  origin {
    domain_name = "bad.s3.amazonaws.com"
    origin_id   = "bad-s3"
    s3_origin_config {
      origin_access_identity = "origin-access-identity/cloudfront/E127EXAMPLE51Z"
    }
  }
}

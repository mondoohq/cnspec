# Two distributions; the second S3 origin has no OAC, so .all() must fail.
resource "aws_cloudfront_distribution" "secure" {
  enabled = true
  origin {
    domain_name              = "good.s3.amazonaws.com"
    origin_id                = "good-s3"
    origin_access_control_id = "E2ABCOAC12345"
    s3_origin_config {
      origin_access_identity = ""
    }
  }
}

resource "aws_cloudfront_distribution" "insecure" {
  enabled = true
  origin {
    domain_name = "bad.s3.amazonaws.com"
    origin_id   = "bad-s3"
    s3_origin_config {
      origin_access_identity = "origin-access-identity/cloudfront/E127EXAMPLE51Z"
    }
  }
}

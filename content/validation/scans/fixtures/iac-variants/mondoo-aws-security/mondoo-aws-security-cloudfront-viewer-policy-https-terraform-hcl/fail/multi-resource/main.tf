# Two distributions; the second allows plain HTTP, so .all() must fail.
resource "aws_cloudfront_distribution" "secure" {
  enabled = true
  default_cache_behavior {
    viewer_protocol_policy = "redirect-to-https"
    target_origin_id       = "origin"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
  }
}

resource "aws_cloudfront_distribution" "insecure" {
  enabled = true
  default_cache_behavior {
    viewer_protocol_policy = "allow-all"
    target_origin_id       = "origin"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
  }
}

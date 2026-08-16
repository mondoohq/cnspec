# Non-compliant: the default CloudFront certificate pins the minimum protocol
# version to TLSv1, and minimum_protocol_version cannot be raised.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

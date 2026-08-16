# Compliant: a custom (non-S3) origin has no s3_origin_config, so the origin
# access control requirement does not apply.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  origin {
    domain_name = "app.example.com"
    origin_id   = "alb-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }
}

# Non-compliant: trust store has no CA certificate bundle source.
resource "aws_cloudfront_trust_store" "example" {
  name = "example"
}

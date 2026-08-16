# Non-compliant: counted trust stores with no CA certificate bundle source.
resource "aws_cloudfront_trust_store" "fleet" {
  count = 2
  name  = "fleet-${count.index}"
}

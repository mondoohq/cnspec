# Two trust stores; the second has no bundle source, so .all() must fail.
resource "aws_cloudfront_trust_store" "with_bundle" {
  name = "with-bundle"
  ca_certificates_bundle_source {
    bucket = "example-bucket"
    key    = "ca-bundle.pem"
  }
}

resource "aws_cloudfront_trust_store" "empty" {
  name = "empty"
}

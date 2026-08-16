# Compliant: trust store references a CA certificate bundle.
resource "aws_cloudfront_trust_store" "example" {
  name = "example"

  ca_certificates_bundle_source {
    bucket  = "example-bucket"
    key     = "ca-bundle.pem"
    version = "1"
  }
}

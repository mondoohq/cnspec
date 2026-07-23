# Compliant: domain encrypted with a referenced customer managed KMS key.
resource "aws_kms_key" "codeartifact" {
  description             = "CodeArtifact domain encryption key"
  deletion_window_in_days = 10
}

resource "aws_codeartifact_domain" "pass_example" {
  domain         = "example-domain"
  encryption_key = aws_kms_key.codeartifact.arn
}

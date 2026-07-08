# Compliant: only a standard aws_kms_key, no external key material.
resource "aws_kms_key" "pass_example" {
  description = "standard key"
}

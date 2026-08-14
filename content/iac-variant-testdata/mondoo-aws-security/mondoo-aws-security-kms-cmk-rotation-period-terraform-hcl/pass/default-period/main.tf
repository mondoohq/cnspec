# Rotation on, period left at the AWS default of 365 days.
resource "aws_kms_key" "data" {
  description         = "Application data key"
  enable_key_rotation = true
}

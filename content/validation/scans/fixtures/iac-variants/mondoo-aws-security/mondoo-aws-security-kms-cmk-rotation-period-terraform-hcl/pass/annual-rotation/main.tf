resource "aws_kms_key" "data" {
  description             = "Application data key"
  enable_key_rotation     = true
  rotation_period_in_days = 365
  deletion_window_in_days = 30
}

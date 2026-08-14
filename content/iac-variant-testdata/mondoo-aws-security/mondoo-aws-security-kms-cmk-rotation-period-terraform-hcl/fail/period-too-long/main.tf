# Rotation is enabled but stretched well past a year, so a compromised key
# stays in service far longer than the control allows.
resource "aws_kms_key" "data" {
  description             = "Application data key"
  enable_key_rotation     = true
  rotation_period_in_days = 2560
}

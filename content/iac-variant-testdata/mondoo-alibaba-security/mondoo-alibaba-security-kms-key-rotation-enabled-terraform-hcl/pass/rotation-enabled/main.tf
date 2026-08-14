resource "alicloud_kms_key" "data" {
  description            = "Application data encryption key"
  key_usage              = "ENCRYPT/DECRYPT"
  automatic_rotation     = "Enabled"
  rotation_interval      = "31536000s"
  pending_window_in_days = 30
}

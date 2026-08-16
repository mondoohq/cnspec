# A long-lived encryption key that never rotates widens the blast radius of a
# key compromise indefinitely.
resource "alicloud_kms_key" "data" {
  description        = "Application data encryption key"
  key_usage          = "ENCRYPT/DECRYPT"
  automatic_rotation = "Disabled"
}

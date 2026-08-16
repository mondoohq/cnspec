# Violation: key ring is created in the "global" location.
resource "google_kms_key_ring" "fail_example" {
  name     = "app-ring"
  location = "global"
}

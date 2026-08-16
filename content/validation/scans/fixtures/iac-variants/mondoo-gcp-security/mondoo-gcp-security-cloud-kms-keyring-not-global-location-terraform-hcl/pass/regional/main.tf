# Compliant: key ring is created in a regional location.
resource "google_kms_key_ring" "pass_example" {
  name     = "app-ring"
  location = "us-central1"
}

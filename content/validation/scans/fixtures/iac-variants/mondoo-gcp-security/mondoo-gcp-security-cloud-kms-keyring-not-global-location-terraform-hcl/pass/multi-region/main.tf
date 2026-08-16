# Compliant: key ring uses a multi-region location (not "global").
resource "google_kms_key_ring" "pass_example" {
  name     = "app-ring"
  location = "us"
}

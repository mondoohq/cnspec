# Compliant: retention policy is locked with a positive retention period.
resource "google_storage_bucket" "wormlike" {
  name     = "my-compliance-bucket"
  location = "US"

  retention_policy {
    is_locked        = true
    retention_period = 2592000
  }
}

# Non-compliant: retention policy exists but is not locked.
resource "google_storage_bucket" "wormlike" {
  name     = "my-compliance-bucket"
  location = "US"

  retention_policy {
    is_locked        = false
    retention_period = 2592000
  }
}

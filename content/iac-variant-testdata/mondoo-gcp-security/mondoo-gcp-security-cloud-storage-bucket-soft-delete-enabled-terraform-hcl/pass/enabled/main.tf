# Compliant: soft delete retention is enabled for 7 days.
resource "google_storage_bucket" "recoverable" {
  name     = "my-recoverable-bucket"
  location = "US"

  soft_delete_policy {
    retention_duration_seconds = 604800
  }
}

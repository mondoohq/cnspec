# Non-compliant: soft delete is explicitly disabled with zero retention.
resource "google_storage_bucket" "nodelete" {
  name     = "my-bucket"
  location = "US"

  soft_delete_policy {
    retention_duration_seconds = 0
  }
}

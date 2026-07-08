# Non-compliant: uniform bucket-level access is not set.
resource "google_storage_bucket" "default" {
  name     = "my-default-bucket"
  location = "US"
}

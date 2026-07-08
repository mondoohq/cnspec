# Non-compliant: no soft delete policy block present.
resource "google_storage_bucket" "default" {
  name     = "my-default-bucket"
  location = "US"
}

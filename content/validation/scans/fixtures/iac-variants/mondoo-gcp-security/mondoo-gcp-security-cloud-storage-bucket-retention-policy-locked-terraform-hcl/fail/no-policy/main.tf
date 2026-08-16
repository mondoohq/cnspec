# Non-compliant: no retention policy configured.
resource "google_storage_bucket" "default" {
  name     = "my-default-bucket"
  location = "US"
}

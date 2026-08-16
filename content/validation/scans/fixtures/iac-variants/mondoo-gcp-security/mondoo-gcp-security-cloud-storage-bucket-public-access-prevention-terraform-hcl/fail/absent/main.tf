# Non-compliant: public access prevention is not configured.
resource "google_storage_bucket" "default" {
  name     = "my-default-bucket"
  location = "US"
}

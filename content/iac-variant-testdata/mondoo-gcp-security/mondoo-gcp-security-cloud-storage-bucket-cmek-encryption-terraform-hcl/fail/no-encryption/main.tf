# Non-compliant: no CMEK encryption block, uses Google-managed keys.
resource "google_storage_bucket" "default" {
  name     = "my-default-bucket"
  location = "US"
}

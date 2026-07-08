# Compliant: uniform bucket-level access is enabled.
resource "google_storage_bucket" "uniform" {
  name                        = "my-uniform-bucket"
  location                    = "US"
  uniform_bucket_level_access = true
}

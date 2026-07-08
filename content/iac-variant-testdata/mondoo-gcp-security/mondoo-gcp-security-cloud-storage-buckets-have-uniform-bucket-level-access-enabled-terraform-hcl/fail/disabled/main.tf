# Non-compliant: uniform bucket-level access is disabled (ACLs allowed).
resource "google_storage_bucket" "legacy" {
  name                        = "my-legacy-bucket"
  location                    = "US"
  uniform_bucket_level_access = false
}

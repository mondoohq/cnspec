# Compliant: bucket encrypted with a customer-managed key.
resource "google_storage_bucket" "cmek" {
  name     = "my-encrypted-bucket"
  location = "US"

  encryption {
    default_kms_key_name = "projects/my-project/locations/us/keyRings/my-ring/cryptoKeys/my-key"
  }
}

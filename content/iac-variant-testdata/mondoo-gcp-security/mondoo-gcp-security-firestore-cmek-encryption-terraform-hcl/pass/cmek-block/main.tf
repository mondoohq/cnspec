# Compliant: the Firestore database has a cmek_config block with a KMS key.
resource "google_firestore_database" "database" {
  project     = "my-project"
  name        = "my-database"
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"

  cmek_config {
    kms_key_name = "projects/my-project/locations/nam5/keyRings/my-ring/cryptoKeys/my-key"
  }
}

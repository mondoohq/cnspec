# Non-compliant: cmek_config block present but kms_key_name is empty.
resource "google_firestore_database" "database" {
  project     = "my-project"
  name        = "my-database"
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"

  cmek_config {
    kms_key_name = ""
  }
}

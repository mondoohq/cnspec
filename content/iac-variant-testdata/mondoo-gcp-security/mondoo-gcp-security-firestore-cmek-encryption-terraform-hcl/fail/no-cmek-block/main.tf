# Non-compliant: no cmek_config block, so Google-managed keys are used.
resource "google_firestore_database" "database" {
  project     = "my-project"
  name        = "my-database"
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"
}

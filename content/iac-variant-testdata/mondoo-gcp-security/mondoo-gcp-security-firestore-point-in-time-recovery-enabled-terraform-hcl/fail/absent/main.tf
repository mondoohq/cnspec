# Non-compliant: point_in_time_recovery_enablement is not set (defaults to disabled).
resource "google_firestore_database" "database" {
  project     = "my-project"
  name        = "my-database"
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"
}

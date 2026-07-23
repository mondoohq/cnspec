# Non-compliant: point-in-time recovery is explicitly disabled.
resource "google_firestore_database" "database" {
  project                           = "my-project"
  name                              = "my-database"
  location_id                       = "nam5"
  type                              = "FIRESTORE_NATIVE"
  point_in_time_recovery_enablement = "POINT_IN_TIME_RECOVERY_DISABLED"
}

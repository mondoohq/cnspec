# Compliant: the schedule inherits encryption from a database that uses a customer-managed key.
resource "google_spanner_database" "cmek" {
  instance = "my-instance"
  name     = "my-database"

  encryption_config {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}

resource "google_spanner_backup_schedule" "pass_example" {
  instance = "my-instance"
  database = google_spanner_database.cmek.name
  name     = "my-schedule"

  retention_duration = "172800s"

  spec {
    cron_spec {
      text = "0 12 * * *"
    }
  }

  full_backup_spec {}

  encryption_config {
    encryption_type = "USE_DATABASE_ENCRYPTION"
  }
}

# Compliant: backup schedule uses a customer-managed encryption key.
resource "google_spanner_backup_schedule" "pass_example" {
  instance = "my-instance"
  database = "my-database"
  name     = "my-schedule"

  retention_duration = "172800s"

  spec {
    cron_spec {
      text = "0 12 * * *"
    }
  }

  full_backup_spec {}

  encryption_config {
    encryption_type = "CUSTOMER_MANAGED_ENCRYPTION"
    kms_key_name    = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}

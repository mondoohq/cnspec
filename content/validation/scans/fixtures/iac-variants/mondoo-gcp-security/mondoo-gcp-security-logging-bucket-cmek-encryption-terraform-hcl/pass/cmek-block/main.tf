# Compliant: log bucket has a cmek_settings block with a KMS key.
resource "google_logging_project_bucket_config" "pass_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 90

  cmek_settings {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/logs/cryptoKeys/logs-key"
  }
}

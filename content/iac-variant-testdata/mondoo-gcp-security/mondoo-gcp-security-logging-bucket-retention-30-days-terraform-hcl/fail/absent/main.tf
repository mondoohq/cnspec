# Non-compliant: retention_days is not set, so it defaults to 30 in GCP but the
# configuration does not explicitly enforce a minimum retention.
resource "google_logging_project_bucket_config" "fail_example" {
  project   = "my-project"
  location  = "us-central1"
  bucket_id = "audit-logs"
}

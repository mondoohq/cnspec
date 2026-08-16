# Non-compliant: retention is below the 30-day minimum.
resource "google_logging_project_bucket_config" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 7
}

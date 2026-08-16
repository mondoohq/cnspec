# Non-compliant: log bucket is explicitly not locked.
resource "google_logging_project_bucket_config" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 365
  locked         = false
}

# Non-compliant: locked is not set, so the bucket defaults to unlocked.
resource "google_logging_project_bucket_config" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 365
}

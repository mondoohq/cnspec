# Non-compliant: log bucket has no cmek_settings block (Google-managed keys).
resource "google_logging_project_bucket_config" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 90
}

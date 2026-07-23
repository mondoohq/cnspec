# Compliant: retention meets the 30-day minimum exactly.
resource "google_logging_project_bucket_config" "pass_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 30
}

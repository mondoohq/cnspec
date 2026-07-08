# Compliant: log bucket is locked, preventing deletion and retention changes.
resource "google_logging_project_bucket_config" "pass_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "audit-logs"
  retention_days = 365
  locked         = true
}

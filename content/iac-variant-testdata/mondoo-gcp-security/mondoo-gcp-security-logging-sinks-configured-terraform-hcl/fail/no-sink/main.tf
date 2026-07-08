# Non-compliant: the configuration manages logging infrastructure but defines no
# google_logging_project_sink, so logs are not exported anywhere.
resource "google_storage_bucket" "audit_logs" {
  name          = "my-project-audit-logs"
  location      = "US"
  force_destroy = false

  uniform_bucket_level_access = true
}

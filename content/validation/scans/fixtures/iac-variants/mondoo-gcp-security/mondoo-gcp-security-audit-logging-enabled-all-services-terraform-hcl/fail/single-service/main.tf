# Non-compliant: audit logging only configured for one service, not allServices.
resource "google_project_iam_audit_config" "storage_only" {
  project = "my-project"
  service = "storage.googleapis.com"

  audit_log_config {
    log_type = "ADMIN_READ"
  }
}

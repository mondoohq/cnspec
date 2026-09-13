# Non-compliant: an administrator-created log bucket in the global location can
# never carry a CMEK key and has to be recreated in a region.
resource "google_logging_project_bucket_config" "fail_example" {
  project        = "my-project"
  location       = "global"
  bucket_id      = "audit-logs"
  retention_days = 90
}

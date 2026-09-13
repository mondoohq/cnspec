# Compliant: the _Required and _Default buckets sit in the global location, where
# Cloud Logging offers no CMEK, so they are out of scope for this check.
resource "google_logging_project_bucket_config" "required" {
  project   = "my-project"
  location  = "global"
  bucket_id = "_Required"
}

resource "google_logging_project_bucket_config" "default" {
  project        = "my-project"
  location       = "global"
  bucket_id      = "_Default"
  retention_days = 30
}

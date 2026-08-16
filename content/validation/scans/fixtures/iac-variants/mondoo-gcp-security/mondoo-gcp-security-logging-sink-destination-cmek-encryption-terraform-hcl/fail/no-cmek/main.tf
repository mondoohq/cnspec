# Non-compliant: log bucket destination has no cmek_settings block.
resource "google_logging_project_bucket_config" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  bucket_id      = "sink-destination"
  retention_days = 90
}

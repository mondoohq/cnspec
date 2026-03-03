# Create a Cloud Logging sink for log export
resource "google_logging_project_sink" "audit_log_sink" {
  name        = "audit-log-sink"
  destination = "storage.googleapis.com/${google_storage_bucket.log_bucket.name}"

  filter = "logName:\"logs/cloudaudit.googleapis.com\""

  unique_writer_identity = true
}

# Create a Cloud Storage bucket for log storage
resource "google_storage_bucket" "log_bucket" {
  name          = "audit-logs-${random_id.rnd.hex}"
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = true

  retention_policy {
    retention_period = 2592000 # 30 days in seconds
  }
}

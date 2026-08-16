# Define the provider configuration
provider "google" {
  project = var.gcp_project_id
  region  = var.region
}

resource "google_logging_project_sink" "audit_sink" {
  name                   = "audit-log-sink"
  destination            = "storage.googleapis.com/audit-logs-bucket"
  filter                 = "logName:\"cloudaudit.googleapis.com\""
  unique_writer_identity = true
}


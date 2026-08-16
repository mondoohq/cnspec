# Compliant: a project-level log sink is configured.
resource "google_logging_project_sink" "pass_example" {
  name        = "audit-sink"
  destination = "storage.googleapis.com/my-project-audit-logs"

  unique_writer_identity = true
}

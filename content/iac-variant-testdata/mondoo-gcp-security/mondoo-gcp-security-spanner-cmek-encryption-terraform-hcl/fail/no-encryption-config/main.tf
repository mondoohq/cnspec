# Non-compliant: no encryption_config, so Google-managed keys are used.
resource "google_spanner_database" "fail_example" {
  instance = "my-instance"
  name     = "my-database"
}

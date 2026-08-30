# With the service_accounts block omitted the instance falls back to the default
# Compute Engine account.
resource "google_workbench_instance" "research" {
  project  = var.project_id
  name     = "research-workbench"
  location = "us-central1-a"

  gce_setup {
    machine_type = "e2-standard-4"
  }
}

# The default Compute Engine service account holds project-wide Editor by
# default, so every notebook on it can rewrite the project.
resource "google_workbench_instance" "research" {
  project  = var.project_id
  name     = "research-workbench"
  location = "us-central1-a"

  gce_setup {
    machine_type = "e2-standard-4"

    service_accounts {
      email = "123456789012-compute@developer.gserviceaccount.com"
    }
  }
}

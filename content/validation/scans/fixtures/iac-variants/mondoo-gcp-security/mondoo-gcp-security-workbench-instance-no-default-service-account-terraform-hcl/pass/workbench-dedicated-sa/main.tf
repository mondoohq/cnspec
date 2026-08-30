resource "google_workbench_instance" "research" {
  project  = var.project_id
  name     = "research-workbench"
  location = "us-central1-a"

  gce_setup {
    machine_type = "e2-standard-4"

    # Purpose-built account scoped to just what the notebook needs.
    service_accounts {
      email = "workbench-research@example-prod.iam.gserviceaccount.com"
    }
  }
}

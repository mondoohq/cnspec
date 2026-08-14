resource "google_notebooks_instance" "research" {
  project      = var.project_id
  name         = "research-workbench"
  location     = "us-central1-a"
  machine_type = "n1-standard-4"

  # Purpose-built account scoped to just what the notebook needs.
  service_account = "workbench-research@example-prod.iam.gserviceaccount.com"

  vm_image {
    project      = "deeplearning-platform-release"
    image_family = "tf-latest-cpu"
  }

  no_public_ip    = true
  no_proxy_access = true
}

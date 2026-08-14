# The default Compute Engine service account holds project-wide Editor by
# default, so every notebook on it can rewrite the project.
resource "google_notebooks_instance" "research" {
  project      = var.project_id
  name         = "research-workbench"
  location     = "us-central1-a"
  machine_type = "n1-standard-4"

  service_account = "123456789012-compute@developer.gserviceaccount.com"

  vm_image {
    project      = "deeplearning-platform-release"
    image_family = "tf-latest-cpu"
  }
}

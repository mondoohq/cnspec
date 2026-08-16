# With service_account omitted the instance silently falls back to the default
# Compute Engine account.
resource "google_notebooks_instance" "research" {
  project      = var.project_id
  name         = "research-workbench"
  location     = "us-central1-a"
  machine_type = "n1-standard-4"

  vm_image {
    project      = "deeplearning-platform-release"
    image_family = "tf-latest-cpu"
  }
}

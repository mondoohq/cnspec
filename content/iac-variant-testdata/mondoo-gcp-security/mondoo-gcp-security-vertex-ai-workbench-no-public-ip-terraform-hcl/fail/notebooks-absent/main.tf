resource "google_notebooks_instance" "fail" {
  name         = "notebooks-instance"
  location     = "us-central1-a"
  machine_type = "e2-medium"
  vm_image {
    project      = "deeplearning-platform-release"
    image_family = "tf-latest-cpu"
  }
}

# Non-compliant: no binary_authorization block, so the cluster admits any image.
resource "google_container_cluster" "fail_absent" {
  name     = "fail-absent"
  location = "us-central1"
}

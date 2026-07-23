resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  ip_allocation_policy {}
}

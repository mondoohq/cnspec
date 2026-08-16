resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }
}

resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  node_pool {
    name = "default-pool"

    node_config {
      machine_type = "e2-medium"

      shielded_instance_config {
        enable_secure_boot          = false
        enable_integrity_monitoring = true
      }
    }
  }
}

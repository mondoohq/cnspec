# Compliant: workload_identity_config binds the cluster to a workload pool.
resource "google_container_cluster" "primary" {
  name     = "wi-cluster"
  location = "us-central1"

  initial_node_count = 1

  workload_identity_config {
    workload_pool = "my-project.svc.id.goog"
  }
}

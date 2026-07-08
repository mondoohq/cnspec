resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  node_config {
    service_account = "gke-node-sa@my-project.iam.gserviceaccount.com"
    oauth_scopes    = ["https://www.googleapis.com/auth/cloud-platform"]
  }
}

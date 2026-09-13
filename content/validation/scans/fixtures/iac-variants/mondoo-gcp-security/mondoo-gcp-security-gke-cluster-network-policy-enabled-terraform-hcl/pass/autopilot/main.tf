resource "google_container_cluster" "autopilot" {
  name     = "autopilot-cluster"
  location = "us-central1"

  enable_autopilot = true
}

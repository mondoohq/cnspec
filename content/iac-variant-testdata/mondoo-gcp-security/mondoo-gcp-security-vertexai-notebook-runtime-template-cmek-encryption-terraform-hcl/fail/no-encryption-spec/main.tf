# Without encryption_spec the runtime's disks fall back to Google-managed keys.
resource "google_colab_runtime_template" "research" {
  provider     = google-beta
  name         = "research-template"
  display_name = "Research notebooks"
  location     = "us-central1"

  machine_spec {
    machine_type = "n1-standard-4"
  }

  network_spec {
    enable_internet_access = false
    network                = "projects/example-prod/global/networks/research-vpc"
  }
}

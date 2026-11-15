# Non-compliant: attached to a VPC but internet access is still enabled.
resource "google_colab_runtime_template" "research" {
  provider     = google-beta
  name         = "research-template"
  display_name = "Research notebooks"
  location     = "us-central1"

  machine_spec {
    machine_type = "n1-standard-4"
  }

  network_spec {
    enable_internet_access = true
    network                = "projects/example-prod/global/networks/research-vpc"
    subnetwork             = "projects/example-prod/regions/us-central1/subnetworks/research"
  }
}

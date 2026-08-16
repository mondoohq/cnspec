# With no network_spec the runtime lands on the default Google-managed network
# instead of a VPC the organization controls.
resource "google_colab_runtime_template" "research" {
  provider     = google-beta
  name         = "research-template"
  display_name = "Research notebooks"
  location     = "us-central1"

  machine_spec {
    machine_type = "n1-standard-4"
  }
}

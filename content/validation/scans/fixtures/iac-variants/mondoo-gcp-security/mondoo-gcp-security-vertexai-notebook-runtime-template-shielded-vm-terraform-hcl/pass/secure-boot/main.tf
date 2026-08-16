resource "google_colab_runtime_template" "research" {
  provider     = google-beta
  name         = "research-template"
  display_name = "Research notebooks"
  location     = "us-central1"

  machine_spec {
    machine_type = "n1-standard-4"
  }

  shielded_vm_config {
    enable_secure_boot = true
  }
}

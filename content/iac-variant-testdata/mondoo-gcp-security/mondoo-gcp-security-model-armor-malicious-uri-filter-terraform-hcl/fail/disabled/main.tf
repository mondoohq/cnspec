# Non-compliant: malicious URI filter present but disabled.
resource "google_model_armor_template" "insecure" {
  template_id = "insecure-template"
  location    = "us-central1"

  filter_config {
    malicious_uri_filter_settings {
      filter_enforcement = "DISABLED"
    }
  }
}

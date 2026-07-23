# Compliant: no rai_settings block, so no HIGH-only filter exists.
resource "google_model_armor_template" "secure" {
  template_id = "secure-template"
  location    = "us-central1"

  filter_config {
    malicious_uri_filter_settings {
      filter_enforcement = "ENABLED"
    }
  }
}

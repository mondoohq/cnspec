# Non-compliant: filter_config present but no pi_and_jailbreak_filter_settings.
resource "google_model_armor_template" "insecure" {
  template_id = "insecure-template"
  location    = "us-central1"

  filter_config {
    malicious_uri_filter_settings {
      filter_enforcement = "ENABLED"
    }
  }
}

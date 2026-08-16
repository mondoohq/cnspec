# Non-compliant: sanitize-operation logging explicitly disabled.
resource "google_model_armor_template" "insecure" {
  template_id = "insecure-template"
  location    = "us-central1"

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "ENABLED"
      confidence_level   = "LOW_AND_ABOVE"
    }
  }

  template_metadata {
    log_sanitize_operations = false
  }
}

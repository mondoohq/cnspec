# Non-compliant: prompt-injection and jailbreak filter present but disabled.
resource "google_model_armor_template" "insecure" {
  template_id = "insecure-template"
  location    = "us-central1"

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "DISABLED"
      confidence_level   = "LOW_AND_ABOVE"
    }
  }
}

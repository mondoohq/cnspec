# Compliant: no template_metadata block, so the API default of not ignoring partial failures applies.
resource "google_model_armor_template" "secure" {
  template_id = "secure-template"
  location    = "us-central1"

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "ENABLED"
    }
  }
}

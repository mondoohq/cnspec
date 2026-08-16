# Non-compliant: enforcement only inspects, never blocks.
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
    enforcement_type = "INSPECT_ONLY"
  }
}

# Compliant: partial detector failures are explicitly not ignored.
resource "google_model_armor_template" "secure" {
  template_id = "secure-template"
  location    = "us-central1"

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "ENABLED"
    }
  }

  template_metadata {
    ignore_partial_invocation_failures = false
  }
}

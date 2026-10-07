# Non-compliant: the template ignores partial detector failures, so a detector that fails to run no longer blocks the request.
resource "google_model_armor_template" "insecure" {
  template_id = "insecure-template"
  location    = "us-central1"

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "ENABLED"
    }
  }

  template_metadata {
    ignore_partial_invocation_failures = true
  }
}

# Compliant: sdp_settings binds Sensitive Data Protection templates (advanced mode).
resource "google_model_armor_template" "pass_example" {
  location    = "us-central1"
  template_id = "pass-template"

  filter_config {
    sdp_settings {
      advanced_config {
        inspect_template = "projects/example-project/locations/us-central1/inspectTemplates/example-inspect"
      }
    }
  }
}

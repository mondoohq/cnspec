# Non-compliant: sdp_settings present but basic mode is disabled.
resource "google_model_armor_template" "fail_example" {
  location    = "us-central1"
  template_id = "fail-template"

  filter_config {
    sdp_settings {
      basic_config {
        filter_enforcement = "DISABLED"
      }
    }
  }
}

# Non-compliant: filter_config present but no sdp_settings block.
resource "google_model_armor_template" "fail_example" {
  location    = "us-central1"
  template_id = "fail-template"

  filter_config {
    rai_settings {
      rai_filters {
        filter_type      = "HATE_SPEECH"
        confidence_level = "MEDIUM_AND_ABOVE"
      }
    }
  }
}

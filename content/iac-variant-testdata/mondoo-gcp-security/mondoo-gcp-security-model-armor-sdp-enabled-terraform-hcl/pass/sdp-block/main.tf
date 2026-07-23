# Compliant: filter_config has an sdp_settings block enabling sensitive data protection.
resource "google_model_armor_template" "pass_example" {
  location    = "us-central1"
  template_id = "pass-template"

  filter_config {
    rai_settings {
      rai_filters {
        filter_type      = "HATE_SPEECH"
        confidence_level = "MEDIUM_AND_ABOVE"
      }
    }

    sdp_settings {
      basic_config {
        filter_enforcement = "ENABLED"
      }
    }
  }
}

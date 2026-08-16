# Compliant: RAI filters flag at medium confidence and above (stricter than HIGH-only).
resource "google_model_armor_template" "secure" {
  template_id = "secure-template"
  location    = "us-central1"

  filter_config {
    rai_settings {
      rai_filters {
        filter_type      = "HATE_SPEECH"
        confidence_level = "MEDIUM_AND_ABOVE"
      }
      rai_filters {
        filter_type      = "DANGEROUS"
        confidence_level = "LOW_AND_ABOVE"
      }
    }
  }
}

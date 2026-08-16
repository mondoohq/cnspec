# Non-compliant: rai_settings present but declares no rai_filters.
resource "google_model_armor_template" "insecure" {
  template_id = "insecure-template"
  location    = "us-central1"

  filter_config {
    rai_settings {
    }
  }
}

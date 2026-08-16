# Non-compliant: no filter_config block at all.
resource "google_model_armor_template" "fail_example" {
  location    = "us-central1"
  template_id = "fail-template"
}

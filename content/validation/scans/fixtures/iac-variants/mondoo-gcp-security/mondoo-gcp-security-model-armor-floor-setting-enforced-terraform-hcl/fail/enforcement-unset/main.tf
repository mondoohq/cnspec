# Non-compliant: enforcement omitted, so the floor setting is not enforced.
resource "google_model_armor_floorsetting" "fail_example" {
  parent   = "organizations/123456789"
  location = "global"

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "ENABLED"
      confidence_level   = "MEDIUM_AND_ABOVE"
    }
  }
}

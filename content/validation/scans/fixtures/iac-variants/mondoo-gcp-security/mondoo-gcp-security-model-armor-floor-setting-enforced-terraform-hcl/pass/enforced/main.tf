# Compliant: organization floor setting with enforcement on.
resource "google_model_armor_floorsetting" "pass_example" {
  parent   = "organizations/123456789"
  location = "global"

  enable_floor_setting_enforcement = true

  filter_config {
    pi_and_jailbreak_filter_settings {
      filter_enforcement = "ENABLED"
      confidence_level   = "MEDIUM_AND_ABOVE"
    }
  }
}

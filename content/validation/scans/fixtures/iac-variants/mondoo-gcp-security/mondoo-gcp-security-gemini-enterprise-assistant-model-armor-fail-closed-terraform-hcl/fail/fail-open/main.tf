# Non-compliant: requests pass unscreened when Model Armor is unavailable.
resource "google_discovery_engine_assistant" "fail_example" {
  location      = "global"
  collection_id = "default_collection"
  engine_id     = "assistant-app"
  assistant_id  = "default_assistant"
  display_name  = "Default assistant"

  customer_policy {
    model_armor_config {
      user_prompt_template = "projects/my-project/locations/global/templates/prompts"
      response_template    = "projects/my-project/locations/global/templates/responses"
      failure_mode         = "FAIL_OPEN"
    }
  }
}

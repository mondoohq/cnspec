# Non-compliant: no customer policy, so Model Armor is not used.
resource "google_discovery_engine_assistant" "fail_example" {
  location      = "global"
  collection_id = "default_collection"
  engine_id     = "assistant-app"
  assistant_id  = "default_assistant"
  display_name  = "Default assistant"
}

# Non-compliant: widget allows unauthenticated public access.
resource "google_discovery_engine_widget_config" "fail_example" {
  location         = "global"
  collection_id    = "default_collection"
  engine_id        = "assistant-app"
  widget_config_id = "default_search_widget_config"

  access_settings {
    allow_public_access = true
  }
}

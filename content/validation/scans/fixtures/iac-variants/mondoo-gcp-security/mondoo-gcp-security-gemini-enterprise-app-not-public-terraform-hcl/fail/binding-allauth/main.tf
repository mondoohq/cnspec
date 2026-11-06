# Non-compliant: binding includes allAuthenticatedUsers.
resource "google_discovery_engine_search_engine_iam_binding" "fail_example" {
  engine_id     = "assistant-app"
  collection_id = "default_collection"
  location      = "global"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "allAuthenticatedUsers",
  ]
}

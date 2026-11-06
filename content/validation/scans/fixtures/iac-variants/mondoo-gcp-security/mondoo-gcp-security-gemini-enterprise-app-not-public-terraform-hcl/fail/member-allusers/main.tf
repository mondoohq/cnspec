# Non-compliant: member grants access to allUsers.
resource "google_discovery_engine_search_engine_iam_member" "fail_example" {
  engine_id     = "assistant-app"
  collection_id = "default_collection"
  location      = "global"
  role   = "roles/viewer"
  member = "allUsers"
}

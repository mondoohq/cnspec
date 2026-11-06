# Compliant: access granted to a named group.
resource "google_discovery_engine_search_engine_iam_member" "pass_example" {
  engine_id     = "assistant-app"
  collection_id = "default_collection"
  location      = "global"
  role   = "roles/viewer"
  member = "group:team@example.com"
}

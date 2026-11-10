# Compliant: binding lists only named principals.
resource "google_discovery_engine_search_engine_iam_binding" "pass_example" {
  engine_id     = "assistant-app"
  collection_id = "default_collection"
  location      = "global"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}

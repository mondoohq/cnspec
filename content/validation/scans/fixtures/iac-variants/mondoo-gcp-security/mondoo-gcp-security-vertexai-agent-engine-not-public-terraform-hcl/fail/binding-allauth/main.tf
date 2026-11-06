# Non-compliant: binding includes allAuthenticatedUsers.
resource "google_vertex_ai_reasoning_engine_iam_binding" "fail_example" {
  reasoning_engine = "projects/my-project/locations/us-central1/reasoningEngines/123"
  region           = "us-central1"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "allAuthenticatedUsers",
  ]
}

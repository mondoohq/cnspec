# Compliant: access granted to a named group.
resource "google_vertex_ai_reasoning_engine_iam_member" "pass_example" {
  reasoning_engine = "projects/my-project/locations/us-central1/reasoningEngines/123"
  region           = "us-central1"
  role   = "roles/viewer"
  member = "group:team@example.com"
}

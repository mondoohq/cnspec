# Compliant: binding lists only named principals.
resource "google_vertex_ai_reasoning_engine_iam_binding" "pass_example" {
  reasoning_engine = "projects/my-project/locations/us-central1/reasoningEngines/123"
  region           = "us-central1"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}

# Compliant: binding lists only named principals.
resource "google_agent_identity_auth_provider_iam_binding" "pass_example" {
  auth_provider_id = "ticketing"
  location         = "us-central1"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}

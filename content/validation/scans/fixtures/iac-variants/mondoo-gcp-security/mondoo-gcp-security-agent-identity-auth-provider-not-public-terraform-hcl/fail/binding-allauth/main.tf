# Non-compliant: binding includes allAuthenticatedUsers.
resource "google_agent_identity_auth_provider_iam_binding" "fail_example" {
  auth_provider_id = "ticketing"
  location         = "us-central1"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "allAuthenticatedUsers",
  ]
}

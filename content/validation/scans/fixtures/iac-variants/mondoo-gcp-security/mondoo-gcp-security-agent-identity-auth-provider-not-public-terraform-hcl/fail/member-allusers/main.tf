# Non-compliant: member grants access to allUsers.
resource "google_agent_identity_auth_provider_iam_member" "fail_example" {
  auth_provider_id = "ticketing"
  location         = "us-central1"
  role   = "roles/viewer"
  member = "allUsers"
}

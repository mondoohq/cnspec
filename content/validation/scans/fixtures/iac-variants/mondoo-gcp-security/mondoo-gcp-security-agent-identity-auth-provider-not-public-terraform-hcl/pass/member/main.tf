# Compliant: access granted to a named group.
resource "google_agent_identity_auth_provider_iam_member" "pass_example" {
  auth_provider_id = "ticketing"
  location         = "us-central1"
  role   = "roles/viewer"
  member = "group:team@example.com"
}

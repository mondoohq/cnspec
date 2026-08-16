# Compliant: the default service accounts are disabled outright.
resource "google_project_default_service_accounts" "disable_default" {
  project = "example-project"
  action  = "DISABLE"
}

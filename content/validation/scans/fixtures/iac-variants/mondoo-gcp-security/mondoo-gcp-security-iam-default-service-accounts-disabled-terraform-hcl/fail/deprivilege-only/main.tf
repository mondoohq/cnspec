# Non-compliant: DEPRIVILEGE strips roles but leaves the default accounts enabled.
resource "google_project_default_service_accounts" "deprivilege" {
  project = "example-project"
  action  = "DEPRIVILEGE"
}

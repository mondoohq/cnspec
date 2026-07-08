resource "github_organization_settings" "main" {
  billing_email = "ops@example.com"
  secret_scanning_push_protection_enabled_for_new_repositories = false
}

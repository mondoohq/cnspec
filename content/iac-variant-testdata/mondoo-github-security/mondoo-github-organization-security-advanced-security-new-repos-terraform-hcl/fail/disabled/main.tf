resource "github_organization_settings" "main" {
  billing_email = "ops@example.com"
  advanced_security_enabled_for_new_repositories = false
}

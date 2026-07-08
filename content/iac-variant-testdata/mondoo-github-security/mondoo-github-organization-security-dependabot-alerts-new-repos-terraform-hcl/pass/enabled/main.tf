resource "github_organization_settings" "main" {
  billing_email = "ops@example.com"
  dependabot_alerts_enabled_for_new_repositories = true
}

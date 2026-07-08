resource "github_organization_settings" "main" {
  billing_email = "ops@example.com"
  default_repository_permission = "admin"
}

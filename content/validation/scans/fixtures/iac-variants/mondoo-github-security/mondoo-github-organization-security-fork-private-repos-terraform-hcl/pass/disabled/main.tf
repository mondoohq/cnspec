resource "github_organization_settings" "main" {
  billing_email = "ops@example.com"
  members_can_fork_private_repositories = false
}

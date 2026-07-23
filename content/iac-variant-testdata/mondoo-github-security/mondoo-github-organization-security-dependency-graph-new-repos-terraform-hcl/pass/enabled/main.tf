resource "github_organization_settings" "main" {
  billing_email = "ops@example.com"
  dependency_graph_enabled_for_new_repositories = true
}

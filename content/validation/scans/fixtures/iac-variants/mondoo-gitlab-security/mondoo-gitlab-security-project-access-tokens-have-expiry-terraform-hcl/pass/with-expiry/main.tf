resource "gitlab_project_access_token" "example" {
  project    = "12345"
  name       = "ci-token"
  scopes     = ["read_repository"]
  expires_at = "2025-12-31"
}

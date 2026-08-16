resource "gitlab_project_access_token" "example" {
  project = "12345"
  name    = "ci-token"
  scopes  = ["read_repository"]
}

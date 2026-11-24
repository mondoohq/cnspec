resource "gitlab_project_hook" "example" {
  project = gitlab_project.example.id
  url     = "https://example.com/hook"
  token   = "example-secret-token"
}

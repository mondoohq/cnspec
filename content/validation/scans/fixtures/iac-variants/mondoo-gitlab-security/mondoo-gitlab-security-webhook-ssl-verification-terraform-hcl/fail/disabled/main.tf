resource "gitlab_project_hook" "example" {
  project                 = gitlab_project.example.id
  url                     = "https://example.com/hook"
  push_events             = true
  enable_ssl_verification = false
}

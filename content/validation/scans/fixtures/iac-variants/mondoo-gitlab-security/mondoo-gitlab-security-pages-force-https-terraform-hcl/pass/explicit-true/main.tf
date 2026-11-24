resource "gitlab_project_pages_settings" "example" {
  project     = gitlab_project.example.id
  force_https = true
}

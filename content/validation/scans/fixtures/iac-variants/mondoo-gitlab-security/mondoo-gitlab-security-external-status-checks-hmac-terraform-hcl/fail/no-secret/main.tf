resource "gitlab_project_external_status_check" "example" {
  project_id   = gitlab_project.example.id
  name         = "Compliance check"
  external_url = "https://example.com/status"
}

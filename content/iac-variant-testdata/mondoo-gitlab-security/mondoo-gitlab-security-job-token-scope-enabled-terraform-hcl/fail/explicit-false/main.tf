resource "gitlab_project_job_token_scopes" "example" {
  project            = gitlab_project.example.id
  enabled            = false
  target_project_ids = [gitlab_project.allowed.id]
}

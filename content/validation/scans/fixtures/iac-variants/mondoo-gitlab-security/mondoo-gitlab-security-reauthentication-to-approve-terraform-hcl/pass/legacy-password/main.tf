resource "gitlab_project_level_mr_approvals" "example" {
  project                     = gitlab_project.example.id
  require_password_to_approve = true
}

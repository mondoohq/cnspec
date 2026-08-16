resource "gitlab_project_level_mr_approvals" "example" {
  project                 = gitlab_project.example.id
  reset_approvals_on_push = false
}

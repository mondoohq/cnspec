resource "gitlab_project_level_mr_approvals" "example" {
  project                                        = gitlab_project.example.id
  reset_approvals_on_push                        = true
  disable_overriding_approvers_per_merge_request = true
}

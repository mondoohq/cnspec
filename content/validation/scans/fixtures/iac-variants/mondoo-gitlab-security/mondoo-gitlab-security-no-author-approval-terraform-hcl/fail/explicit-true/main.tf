resource "gitlab_project_level_mr_approvals" "example" {
  project                        = gitlab_project.example.id
  merge_requests_author_approval = true
}

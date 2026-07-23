resource "gitlab_project_push_rules" "example" {
  project                 = gitlab_project.example.id
  reject_unsigned_commits = false
}

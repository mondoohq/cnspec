resource "gitlab_project_push_rules" "example" {
  project                = "12345"
  commit_committer_check = true
}

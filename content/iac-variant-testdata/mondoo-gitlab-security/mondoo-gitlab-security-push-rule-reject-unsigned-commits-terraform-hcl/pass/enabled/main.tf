resource "gitlab_project_push_rules" "example" {
  project                = gitlab_project.example.id
  author_email_regex     = "@example\\.com$"
  commit_committer_check = true
  reject_unsigned_commits = true
}

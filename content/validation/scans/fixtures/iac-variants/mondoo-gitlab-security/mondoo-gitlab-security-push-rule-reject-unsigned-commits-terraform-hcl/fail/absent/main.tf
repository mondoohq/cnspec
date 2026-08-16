resource "gitlab_project_push_rules" "example" {
  project            = gitlab_project.example.id
  author_email_regex = "@example\\.com$"
  prevent_secrets    = true
}

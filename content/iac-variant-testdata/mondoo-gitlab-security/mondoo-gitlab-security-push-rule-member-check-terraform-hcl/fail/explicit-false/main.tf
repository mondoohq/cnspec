resource "gitlab_project_push_rules" "example" {
  project      = "12345"
  member_check = false
}

resource "gitlab_branch_protection" "default" {
  project          = gitlab_project.example.id
  branch           = "main"
  allow_force_push = false
}

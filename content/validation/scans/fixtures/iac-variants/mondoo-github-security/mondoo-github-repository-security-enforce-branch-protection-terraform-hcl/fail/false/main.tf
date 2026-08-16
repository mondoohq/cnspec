resource "github_branch_protection" "main" {
  repository_id  = github_repository.example.node_id
  pattern        = "main"
  enforce_admins = false
}

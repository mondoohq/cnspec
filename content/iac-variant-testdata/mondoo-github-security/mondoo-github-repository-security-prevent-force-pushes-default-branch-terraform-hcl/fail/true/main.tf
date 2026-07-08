resource "github_branch_protection" "main" {
  repository_id       = github_repository.example.node_id
  pattern             = "main"
  allows_force_pushes = true
}

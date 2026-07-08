resource "github_branch_protection" "main" {
  repository_id = github_repository.example.node_id
  pattern       = "main"

  restrict_pushes {
    blocks_creations = true
    push_allowances  = [data.github_user.admin.node_id]
  }
}

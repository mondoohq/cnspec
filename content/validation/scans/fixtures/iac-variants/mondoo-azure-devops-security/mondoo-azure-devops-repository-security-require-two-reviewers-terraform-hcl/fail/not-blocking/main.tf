resource "azuredevops_branch_policy_min_reviewers" "main" {
  project_id = azuredevops_project.example.id
  blocking   = false

  settings {
    reviewer_count = 2

    scope {
      repository_id  = azuredevops_git_repository.example.id
      repository_ref = "refs/heads/main"
      match_type     = "Exact"
    }
  }
}

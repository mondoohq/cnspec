resource "azuredevops_branch_policy_status_check" "main" {
  project_id = azuredevops_project.example.id

  settings {
    name = "security-scan"

    scope {
      repository_id  = azuredevops_git_repository.example.id
      repository_ref = "refs/heads/main"
      match_type     = "Exact"
    }
  }
}

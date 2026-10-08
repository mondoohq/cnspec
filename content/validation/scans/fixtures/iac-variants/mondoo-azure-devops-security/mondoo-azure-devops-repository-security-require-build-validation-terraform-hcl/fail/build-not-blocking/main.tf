resource "azuredevops_branch_policy_build_validation" "main" {
  project_id = azuredevops_project.example.id
  blocking   = false

  settings {
    display_name        = "Pull request build"
    build_definition_id = 1

    scope {
      repository_id  = azuredevops_git_repository.example.id
      repository_ref = "refs/heads/main"
      match_type     = "Exact"
    }
  }
}

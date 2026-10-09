resource "azuredevops_servicehook_webhook_tfs" "main" {
  project_id = azuredevops_project.example.id
  url        = "http://hooks.example.com/azure-devops"

  git_push {
    repository_id = azuredevops_git_repository.example.id
  }
}

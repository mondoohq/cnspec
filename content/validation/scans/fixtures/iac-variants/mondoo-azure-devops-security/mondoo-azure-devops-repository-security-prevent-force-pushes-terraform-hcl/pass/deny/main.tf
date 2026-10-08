resource "azuredevops_git_permissions" "main" {
  project_id    = azuredevops_project.example.id
  repository_id = azuredevops_git_repository.example.id
  principal     = "vssgp.Uy0xLTktMTU1MTM3NDI0NS0xMjM0NTY3ODkw"
  permissions = {
    ForcePush = "Deny"
  }
}

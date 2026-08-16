resource "github_actions_repository_permissions" "example" {
  repository      = github_repository.example.name
  allowed_actions = "all"
}

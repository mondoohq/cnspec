resource "github_actions_repository_permissions" "example" {
  repository          = github_repository.example.name
  allowed_actions     = "selected"
  sha_pinning_required = false

  allowed_actions_config {
    github_owned_allowed = true
    verified_allowed     = true
  }
}

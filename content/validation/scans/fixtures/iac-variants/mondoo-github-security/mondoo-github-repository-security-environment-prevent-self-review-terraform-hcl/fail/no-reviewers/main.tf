resource "github_repository_environment" "prod" {
  repository          = github_repository.example.name
  environment         = "production"
  prevent_self_review = true
}

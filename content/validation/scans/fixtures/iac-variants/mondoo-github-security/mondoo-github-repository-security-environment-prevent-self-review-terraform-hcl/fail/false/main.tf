resource "github_repository_environment" "prod" {
  repository          = github_repository.example.name
  environment         = "production"
  prevent_self_review = false

  reviewers {
    users = [data.github_user.lead.id]
  }
}

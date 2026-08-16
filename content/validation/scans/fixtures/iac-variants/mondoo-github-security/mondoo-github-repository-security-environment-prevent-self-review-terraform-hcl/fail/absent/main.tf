resource "github_repository_environment" "prod" {
  repository  = github_repository.example.name
  environment = "production"

  reviewers {
    users = [data.github_user.lead.id]
  }
}

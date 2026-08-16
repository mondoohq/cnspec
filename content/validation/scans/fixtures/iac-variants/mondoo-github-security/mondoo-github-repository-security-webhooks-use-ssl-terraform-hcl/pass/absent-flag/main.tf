resource "github_repository_webhook" "ci" {
  repository = github_repository.example.name

  configuration {
    url          = "https://ci.example.com/webhook"
    content_type = "json"
  }

  active = true
  events = ["push"]
}

variable "webhook_signing_token" {
  type      = string
  sensitive = true
}

resource "gitlab_project_hook" "example" {
  project       = gitlab_project.example.id
  url           = "https://example.com/hook"
  signing_token = var.webhook_signing_token
}

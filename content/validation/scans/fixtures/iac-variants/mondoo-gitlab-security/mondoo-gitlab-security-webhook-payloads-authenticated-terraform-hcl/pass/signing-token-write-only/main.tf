variable "webhook_signing_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "gitlab_project_hook" "example" {
  project                  = gitlab_project.example.id
  url                      = "https://example.com/hook"
  signing_token_wo         = var.webhook_signing_token
  signing_token_wo_version = "1"
}

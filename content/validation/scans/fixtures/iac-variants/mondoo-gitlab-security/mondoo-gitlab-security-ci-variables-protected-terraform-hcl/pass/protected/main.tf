resource "gitlab_project_variable" "example" {
  project   = gitlab_project.example.id
  key       = "EXAMPLE_TOKEN"
  value     = var.example_token
  protected = true
}

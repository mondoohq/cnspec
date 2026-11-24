resource "github_codespaces_organization_secret" "main" {
  secret_name             = "EXAMPLE_SECRET"
  plaintext_value         = "example"
  visibility              = "selected"
  selected_repository_ids = [123456789]
}

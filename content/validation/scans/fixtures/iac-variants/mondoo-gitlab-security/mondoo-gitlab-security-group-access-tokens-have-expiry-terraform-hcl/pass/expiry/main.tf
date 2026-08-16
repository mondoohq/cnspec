resource "gitlab_group_access_token" "example" {
  group      = gitlab_group.example.id
  name       = "example-token"
  expires_at = "2025-12-31"
  scopes     = ["api"]

  access_level = "maintainer"
}

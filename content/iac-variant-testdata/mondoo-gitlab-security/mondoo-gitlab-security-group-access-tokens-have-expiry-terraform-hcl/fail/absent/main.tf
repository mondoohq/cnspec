resource "gitlab_group_access_token" "example" {
  group  = gitlab_group.example.id
  name   = "example-token"
  scopes = ["api"]

  access_level = "maintainer"
}

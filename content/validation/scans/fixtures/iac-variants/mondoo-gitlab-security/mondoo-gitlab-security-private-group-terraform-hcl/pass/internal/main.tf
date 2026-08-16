resource "gitlab_group" "example" {
  name             = "example"
  path             = "example"
  visibility_level = "internal"
}

resource "gitlab_group" "example" {
  name                          = "example"
  path                          = "example"
  prevent_forking_outside_group = false
}

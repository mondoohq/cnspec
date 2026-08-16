resource "gitlab_group" "example" {
  name                              = "example"
  path                              = "example"
  require_two_factor_authentication = false
}

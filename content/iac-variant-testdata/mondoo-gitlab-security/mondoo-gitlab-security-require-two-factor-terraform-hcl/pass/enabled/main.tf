resource "gitlab_group" "example" {
  name                              = "example"
  path                              = "example"
  require_two_factor_authentication = true
  two_factor_grace_period           = 24
}

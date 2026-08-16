# authentication_method 2 is LDAP (external), not internal.
resource "portainer_settings" "this" {
  authentication_method = 2
}

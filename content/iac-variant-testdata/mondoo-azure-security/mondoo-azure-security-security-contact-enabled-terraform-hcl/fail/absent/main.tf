resource "azurerm_security_center_contact" "fail" {
  name             = "default"
  email            = "security@example.com"
  alerts_to_admins = true
}

resource "azurerm_security_center_contact" "fail" {
  name                = "default"
  email               = "security@example.com"
  alert_notifications = false
  alerts_to_admins    = true
}

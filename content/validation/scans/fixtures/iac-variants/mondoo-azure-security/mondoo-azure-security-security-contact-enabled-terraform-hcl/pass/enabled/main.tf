resource "azurerm_security_center_contact" "pass" {
  name                = "default"
  email               = "security@example.com"
  phone               = "+1-555-555-5555"
  alert_notifications = true
  alerts_to_admins    = true
}

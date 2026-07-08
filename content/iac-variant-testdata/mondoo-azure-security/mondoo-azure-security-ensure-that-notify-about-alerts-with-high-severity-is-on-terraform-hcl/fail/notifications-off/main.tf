resource "azurerm_security_center_contact" "example" {
  name                = "default"
  email               = "security@example.com"
  phone               = "+1-555-0100"
  alert_notifications = false
  alerts_to_admins    = true
}

resource "azurerm_security_center_contact" "fail" {
  name                = "default"
  email               = ""
  alert_notifications = true
  alerts_to_admins    = true
}

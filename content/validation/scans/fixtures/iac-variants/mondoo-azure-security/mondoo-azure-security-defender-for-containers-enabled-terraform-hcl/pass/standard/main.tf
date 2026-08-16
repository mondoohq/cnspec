resource "azurerm_security_center_subscription_pricing" "this" {
  tier          = "Standard"
  resource_type = "Containers"
}

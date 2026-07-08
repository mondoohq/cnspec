resource "azurerm_security_center_subscription_pricing" "mine" {
  tier          = "Standard"
  resource_type = "Containers"
}

resource "azurerm_security_center_subscription_pricing" "other" {
  tier          = "Free"
  resource_type = "StorageAccounts"
}

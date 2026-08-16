resource "azurerm_security_center_subscription_pricing" "bad" {
  tier          = "Free"
  resource_type = "KeyVaults"
}

resource "azurerm_security_center_subscription_pricing" "other" {
  tier          = "Standard"
  resource_type = "StorageAccounts"
}

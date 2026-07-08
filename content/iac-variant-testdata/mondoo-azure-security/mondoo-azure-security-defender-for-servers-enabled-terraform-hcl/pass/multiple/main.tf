resource "azurerm_security_center_subscription_pricing" "mine" {
  tier          = "Standard"
  resource_type = "VirtualMachines"
}

resource "azurerm_security_center_subscription_pricing" "other" {
  tier          = "Free"
  resource_type = "StorageAccounts"
}

resource "azurerm_security_center_subscription_pricing" "storage" {
  tier          = "Free"
  resource_type = "StorageAccounts"
}

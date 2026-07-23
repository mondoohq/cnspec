resource "azurerm_security_center_subscription_pricing" "this" {
  tier          = "Free"
  resource_type = "KeyVaults"
}

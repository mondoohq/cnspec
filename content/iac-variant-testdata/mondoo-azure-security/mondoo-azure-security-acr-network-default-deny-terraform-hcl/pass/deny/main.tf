resource "azurerm_container_registry" "pass" {
  name                = "examplereg"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "Premium"

  network_rule_set {
    default_action = "Deny"
  }
}

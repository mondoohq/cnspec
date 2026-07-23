resource "azurerm_app_configuration" "pass" {
  name                   = "example-appconf"
  resource_group_name    = "example-rg"
  location               = "eastus"
  sku                    = "standard"
  public_network_access  = "Disabled"
}

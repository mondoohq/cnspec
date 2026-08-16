resource "azurerm_databricks_workspace" "example" {
  name                          = "example-databricks"
  resource_group_name           = "example-rg"
  location                      = "eastus"
  sku                           = "premium"
  public_network_access_enabled = false
}

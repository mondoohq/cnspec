resource "azurerm_databricks_workspace" "example" {
  name                = "example-databricks"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "premium"

  custom_parameters {
    no_public_ip = false
  }
}

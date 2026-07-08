resource "azurerm_data_factory" "example" {
  name                    = "example-adf"
  location                = "eastus"
  resource_group_name     = "example-rg"
  customer_managed_key_id = azurerm_key_vault_key.example.id

  identity {
    type = "SystemAssigned"
  }
}

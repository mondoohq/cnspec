resource "azurerm_cognitive_account" "vision" {
  name                = "vision"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  kind                = "ComputerVision"
  sku_name            = "S1"

  identity {
    type = "SystemAssigned"
  }

  customer_managed_key {
    key_vault_key_id = azurerm_key_vault_key.cognitive.id
  }
}

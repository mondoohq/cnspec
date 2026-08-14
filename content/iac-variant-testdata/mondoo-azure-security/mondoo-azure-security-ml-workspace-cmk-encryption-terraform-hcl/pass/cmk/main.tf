resource "azurerm_machine_learning_workspace" "research" {
  name                    = "research-ml"
  location                = azurerm_resource_group.prod.location
  resource_group_name     = azurerm_resource_group.prod.name
  application_insights_id = azurerm_application_insights.ml.id
  key_vault_id            = azurerm_key_vault.prod.id
  storage_account_id      = azurerm_storage_account.ml.id

  identity {
    type = "SystemAssigned"
  }

  encryption {
    key_vault_id = azurerm_key_vault.prod.id
    key_id       = azurerm_key_vault_key.ml.id
  }
}

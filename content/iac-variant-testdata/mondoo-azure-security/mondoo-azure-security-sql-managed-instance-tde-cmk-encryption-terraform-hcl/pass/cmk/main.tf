resource "azurerm_mssql_managed_instance_transparent_data_encryption" "prod" {
  managed_instance_id = azurerm_mssql_managed_instance.prod.id
  key_vault_key_id    = azurerm_key_vault_key.sqlmi.id
}

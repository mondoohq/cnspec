# TDE is on but with the service-managed key, so the account cannot revoke
# access to the data by revoking the key.
resource "azurerm_mssql_managed_instance_transparent_data_encryption" "prod" {
  managed_instance_id = azurerm_mssql_managed_instance.prod.id
}

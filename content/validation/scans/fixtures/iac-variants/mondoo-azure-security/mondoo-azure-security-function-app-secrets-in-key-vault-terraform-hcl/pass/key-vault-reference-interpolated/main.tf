# Compliant: the Key Vault reference is built with an interpolation rather than
# a literal secret URI, which is how a configuration that also manages the vault
# writes it. HCL parses such a value into its template parts instead of a
# string, so the leading part is what carries the @Microsoft.KeyVault( prefix.
resource "azurerm_linux_function_app" "processor" {
  name                = "event-processor"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  service_plan_id     = azurerm_service_plan.prod.id

  storage_account_name       = azurerm_storage_account.functions.name
  storage_account_access_key = azurerm_storage_account.functions.primary_access_key

  app_settings = {
    "FUNCTIONS_WORKER_RUNTIME" = "python"
    "DB_CONNECTIONSTRING"      = "@Microsoft.KeyVault(SecretUri=${azurerm_key_vault_secret.db.versionless_id})"
    "PARTNER_APIKEY"           = "@Microsoft.KeyVault(SecretUri=${azurerm_key_vault_secret.partner_api.versionless_id})"
  }

  site_config {}
}

resource "azurerm_linux_function_app" "processor" {
  name                = "event-processor"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  service_plan_id     = azurerm_service_plan.prod.id
  storage_account_name       = azurerm_storage_account.functions.name
  storage_account_access_key = azurerm_storage_account.functions.primary_access_key

  app_settings = {
    "FUNCTIONS_WORKER_RUNTIME" = "python"
    "DB_CONNECTIONSTRING"      = "@Microsoft.KeyVault(SecretUri=https://example-kv.vault.azure.net/secrets/db-connection/)"
    "PARTNER_APIKEY"           = "@Microsoft.KeyVault(SecretUri=https://example-kv.vault.azure.net/secrets/partner-api-key/)"
  }

  site_config {}
}

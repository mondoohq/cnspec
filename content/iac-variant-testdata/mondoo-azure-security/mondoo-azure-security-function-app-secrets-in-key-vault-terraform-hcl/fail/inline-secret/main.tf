# A literal connection string in app settings is readable by anyone with
# Contributor on the app, and it lands in the Terraform state as well.
resource "azurerm_linux_function_app" "processor" {
  name                = "event-processor"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  service_plan_id     = azurerm_service_plan.prod.id
  storage_account_name       = azurerm_storage_account.functions.name
  storage_account_access_key = azurerm_storage_account.functions.primary_access_key

  app_settings = {
    "FUNCTIONS_WORKER_RUNTIME" = "python"
    "DB_CONNECTIONSTRING"      = "Server=tcp:example.database.windows.net;User ID=app;Password=P@ssw0rd-not-a-real-secret;"
  }

  site_config {}
}

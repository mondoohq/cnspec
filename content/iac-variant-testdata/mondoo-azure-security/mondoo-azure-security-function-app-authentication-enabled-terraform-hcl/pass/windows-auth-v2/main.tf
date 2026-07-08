resource "azurerm_windows_function_app" "example" {
  name                       = "example-func"
  resource_group_name        = "example-rg"
  location                   = "eastus"
  service_plan_id            = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"
  storage_account_name       = "examplestorage"
  storage_account_access_key = "examplekey=="

  site_config {}

  auth_settings_v2 {
    auth_enabled           = true
    unauthenticated_action = "RedirectToLoginPage"

    login {}
  }
}

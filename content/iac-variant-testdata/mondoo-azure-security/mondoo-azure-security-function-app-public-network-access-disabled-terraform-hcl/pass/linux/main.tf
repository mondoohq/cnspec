resource "azurerm_linux_function_app" "pass" {
  name                = "example-linux-function-app"
  resource_group_name = "example-rg"
  location            = "eastus"

  storage_account_name       = "examplesa"
  storage_account_access_key = "exampleaccesskey"
  service_plan_id            = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"

  public_network_access_enabled = false

  site_config {
    application_stack {
      node_version = "18"
    }
  }
}

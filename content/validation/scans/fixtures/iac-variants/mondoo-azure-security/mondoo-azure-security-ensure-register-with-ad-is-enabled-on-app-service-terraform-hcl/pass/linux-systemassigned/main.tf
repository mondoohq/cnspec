resource "azurerm_linux_web_app" "example" {
  name                = "example-linux-app"
  resource_group_name = "example-rg"
  location            = "eastus"
  service_plan_id     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Web/serverfarms/plan"

  site_config {}

  identity {
    type = "SystemAssigned"
  }
}

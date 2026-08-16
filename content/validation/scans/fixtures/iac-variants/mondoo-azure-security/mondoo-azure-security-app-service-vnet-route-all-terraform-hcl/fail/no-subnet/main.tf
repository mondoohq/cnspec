resource "azurerm_windows_web_app" "fail" {
  name                = "example-windows-app"
  location            = "eastus"
  resource_group_name = "example-rg"
  service_plan_id     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"

  site_config {
    vnet_route_all_enabled = true
  }
}

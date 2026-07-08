resource "azurerm_windows_web_app" "example" {
  name                = "example-windows-web-app"
  resource_group_name = "example-rg"
  location            = "eastus"
  service_plan_id     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"

  site_config {
    minimum_tls_version = "1.2"
  }
}

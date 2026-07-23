resource "azurerm_linux_web_app" "example" {
  name                = "example-linux-app"
  location            = "eastus"
  resource_group_name = "example-rg"
  service_plan_id     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"

  client_certificate_enabled = false

  site_config {}
}

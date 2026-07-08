resource "azurerm_linux_web_app" "example" {
  name                = "example-web-app"
  resource_group_name = "example-rg"
  location            = "eastus"
  service_plan_id     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"

  ftp_publish_basic_authentication_enabled = true

  site_config {}
}

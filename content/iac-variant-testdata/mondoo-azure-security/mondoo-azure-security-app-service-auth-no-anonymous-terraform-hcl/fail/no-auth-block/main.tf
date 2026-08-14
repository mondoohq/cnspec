# No auth_settings_v2 at all: the platform performs no authentication.
resource "azurerm_windows_web_app" "portal" {
  name                = "example-portal"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  service_plan_id     = azurerm_service_plan.prod.id

  site_config {}
}

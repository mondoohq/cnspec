# Authentication is configured but anonymous requests are still served, so the
# app's own code is the only thing standing between the internet and its data.
resource "azurerm_linux_web_app" "portal" {
  name                = "example-portal"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  service_plan_id     = azurerm_service_plan.prod.id

  site_config {}

  auth_settings_v2 {
    auth_enabled           = true
    require_authentication = false
    unauthenticated_action = "AllowAnonymous"

    login {}
  }
}

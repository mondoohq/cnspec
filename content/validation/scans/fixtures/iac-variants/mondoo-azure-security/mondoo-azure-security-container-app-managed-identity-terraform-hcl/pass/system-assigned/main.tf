resource "azurerm_container_app" "api" {
  name                         = "api"
  container_app_environment_id = azurerm_container_app_environment.prod.id
  resource_group_name          = azurerm_resource_group.prod.name
  revision_mode                = "Single"

  identity {
    type = "SystemAssigned"
  }

  template {
    container {
      name   = "api"
      image  = "example.azurecr.io/api:1.4.2"
      cpu    = 0.5
      memory = "1Gi"
    }
  }
}

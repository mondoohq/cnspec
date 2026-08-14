# Internal-only ingress needs no IP allow list: it is not reachable from
# outside the container app environment.
resource "azurerm_container_app" "worker" {
  name                         = "worker"
  container_app_environment_id = azurerm_container_app_environment.prod.id
  resource_group_name          = azurerm_resource_group.prod.name
  revision_mode                = "Single"

  ingress {
    external_enabled = false
    target_port      = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    container {
      name   = "worker"
      image  = "example.azurecr.io/worker:1.4.2"
      cpu    = 0.5
      memory = "1Gi"
    }
  }
}

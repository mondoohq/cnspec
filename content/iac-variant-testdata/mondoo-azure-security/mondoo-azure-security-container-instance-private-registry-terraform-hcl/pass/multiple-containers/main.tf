resource "azurerm_container_group" "example" {
  name                = "example-container-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

  container {
    name   = "app"
    image  = "prodregistry.azurecr.io/frontend:v2.1.0"
    cpu    = "0.5"
    memory = "1.5"

    ports {
      port     = 443
      protocol = "TCP"
    }
  }

  container {
    name   = "sidecar"
    image  = "prodregistry.azurecr.io/logging-agent:latest"
    cpu    = "0.25"
    memory = "0.5"
  }
}

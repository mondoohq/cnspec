resource "azurerm_container_group" "compliant" {
  name                = "compliant-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

  container {
    name   = "app"
    image  = "prodregistry.azurecr.io/frontend:v2.1.0"
    cpu    = "0.5"
    memory = "1.5"
  }
}

resource "azurerm_container_group" "violating" {
  name                = "violating-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

  container {
    name   = "app"
    image  = "nginx:latest"
    cpu    = "0.5"
    memory = "1.5"
  }
}

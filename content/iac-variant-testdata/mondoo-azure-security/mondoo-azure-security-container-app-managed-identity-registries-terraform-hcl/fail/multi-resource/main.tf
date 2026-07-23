resource "azurerm_container_app" "compliant" {
  name                         = "compliant-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  identity {
    type = "SystemAssigned"
  }

  registry {
    server   = "myregistry.azurecr.io"
    identity = "System"
  }

  template {
    container {
      name   = "app"
      image  = "myregistry.azurecr.io/app@sha256:abc123"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }
}

resource "azurerm_container_app" "violating" {
  name                         = "violating-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  secret {
    name  = "registry-password"
    value = "s3cr3t"
  }

  registry {
    server               = "myregistry.azurecr.io"
    username             = "admin"
    password_secret_name = "registry-password"
  }

  template {
    container {
      name   = "app"
      image  = "myregistry.azurecr.io/app@sha256:abc123"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }
}

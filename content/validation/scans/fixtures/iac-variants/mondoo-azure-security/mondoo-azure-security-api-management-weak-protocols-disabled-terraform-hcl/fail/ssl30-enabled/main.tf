# SSL 3.0 is broken by POODLE and has been deprecated for a decade; enabling it
# on the frontend downgrades every client that will negotiate it.
resource "azurerm_api_management" "internal" {
  name                = "internal-apim"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  publisher_name      = "Example Corp"
  publisher_email     = "platform@example.com"
  sku_name            = "Developer_1"

  security {
    frontend_ssl30_enabled     = true
    backend_ssl30_enabled      = false
    triple_des_ciphers_enabled = false
  }
}

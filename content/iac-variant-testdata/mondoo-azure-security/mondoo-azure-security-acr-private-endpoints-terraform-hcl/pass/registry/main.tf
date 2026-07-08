resource "azurerm_private_endpoint" "pass" {
  name                = "example-pe"
  location            = "eastus"
  resource_group_name = "example-rg"
  subnet_id           = "/subscriptions/x/subnet"

  private_service_connection {
    name                           = "example-psc"
    private_connection_resource_id = "/subscriptions/x/registries/examplereg"
    is_manual_connection           = false
    subresource_names              = ["registry"]
  }
}

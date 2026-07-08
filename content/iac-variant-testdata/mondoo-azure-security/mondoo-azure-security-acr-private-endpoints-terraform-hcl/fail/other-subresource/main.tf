resource "azurerm_private_endpoint" "fail" {
  name                = "example-pe"
  location            = "eastus"
  resource_group_name = "example-rg"
  subnet_id           = "/subscriptions/x/subnet"

  private_service_connection {
    name                           = "example-psc"
    private_connection_resource_id = "/subscriptions/x/accounts/examplestor"
    is_manual_connection           = false
    subresource_names              = ["blob"]
  }
}

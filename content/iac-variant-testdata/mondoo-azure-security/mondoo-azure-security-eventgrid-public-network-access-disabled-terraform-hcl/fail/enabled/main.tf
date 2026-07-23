resource "azurerm_eventgrid_topic" "example" {
  name                          = "example-topic"
  location                      = "eastus"
  resource_group_name           = "example-rg"
  public_network_access_enabled = true
}

resource "azurerm_batch_account" "example" {
  name                          = "examplebatchaccount"
  resource_group_name           = "example-rg"
  location                      = "eastus"
  public_network_access_enabled = false
}

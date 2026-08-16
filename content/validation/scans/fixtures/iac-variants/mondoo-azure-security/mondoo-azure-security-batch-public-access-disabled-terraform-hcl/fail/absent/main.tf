resource "azurerm_batch_account" "example" {
  name                = "examplebatchaccount"
  resource_group_name = "example-rg"
  location            = "eastus"
}

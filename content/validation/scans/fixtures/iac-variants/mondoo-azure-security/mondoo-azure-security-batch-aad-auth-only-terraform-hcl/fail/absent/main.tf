resource "azurerm_batch_account" "fail" {
  name                = "examplebatch"
  resource_group_name = "example-rg"
  location            = "eastus"

}

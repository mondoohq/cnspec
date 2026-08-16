resource "azurerm_batch_account" "pass" {
  name                = "examplebatch"
  resource_group_name = "example-rg"
  location            = "eastus"
  allowed_authentication_modes = ["AAD"]
}

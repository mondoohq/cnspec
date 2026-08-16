resource "azurerm_automation_account" "fail" {
  name                = "example-automation"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku_name            = "Basic"

}

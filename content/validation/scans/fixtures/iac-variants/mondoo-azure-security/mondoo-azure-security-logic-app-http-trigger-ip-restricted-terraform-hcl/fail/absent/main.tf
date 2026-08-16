resource "azurerm_logic_app_workflow" "example" {
  name                = "example-workflow"
  location            = "eastus"
  resource_group_name = "example-rg"
}

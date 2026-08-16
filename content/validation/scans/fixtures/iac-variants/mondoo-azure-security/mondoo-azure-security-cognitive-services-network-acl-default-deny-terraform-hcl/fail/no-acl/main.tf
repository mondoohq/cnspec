resource "azurerm_cognitive_account" "example" {
  name                = "example-cognitive"
  location            = "eastus"
  resource_group_name = "example-rg"
  kind                = "OpenAI"
  sku_name            = "S0"
}

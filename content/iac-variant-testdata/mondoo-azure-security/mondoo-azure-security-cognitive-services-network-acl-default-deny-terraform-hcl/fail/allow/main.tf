resource "azurerm_cognitive_account" "example" {
  name                = "example-cognitive"
  location            = "eastus"
  resource_group_name = "example-rg"
  kind                = "OpenAI"
  sku_name            = "S0"

  network_acls {
    default_action = "Allow"
    ip_rules       = ["203.0.113.0/24"]
  }
}

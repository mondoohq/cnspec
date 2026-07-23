resource "azurerm_user_assigned_identity" "example" {
  name                = "example-uai"
  location            = "eastus"
  resource_group_name = "example-rg"
}

resource "azurerm_cognitive_account" "example" {
  name                = "example-cognitive"
  location            = "eastus"
  resource_group_name = "example-rg"
  kind                = "OpenAI"
  sku_name            = "S0"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.example.id]
  }
}

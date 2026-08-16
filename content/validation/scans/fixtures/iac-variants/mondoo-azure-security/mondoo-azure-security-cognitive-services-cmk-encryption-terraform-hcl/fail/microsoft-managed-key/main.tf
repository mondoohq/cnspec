# Without a customer_managed_key block the account's stored data, including
# uploaded training and inference content, uses the Microsoft-managed key.
resource "azurerm_cognitive_account" "vision" {
  name                = "vision"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  kind                = "ComputerVision"
  sku_name            = "S1"
}

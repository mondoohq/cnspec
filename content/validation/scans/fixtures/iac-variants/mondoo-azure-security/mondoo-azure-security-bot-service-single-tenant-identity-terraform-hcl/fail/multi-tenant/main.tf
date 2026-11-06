resource "azurerm_bot_service_azure_bot" "example" {
  name                = "example-bot"
  resource_group_name = "rg"
  location            = "global"
  sku                 = "S1"
  microsoft_app_id    = "00000000-0000-0000-0000-000000000000"
  microsoft_app_type      = "MultiTenant"
}

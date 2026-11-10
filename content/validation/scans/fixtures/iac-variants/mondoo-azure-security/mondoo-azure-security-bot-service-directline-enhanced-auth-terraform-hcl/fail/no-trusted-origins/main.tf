resource "azurerm_bot_channel_directline" "example" {
  bot_name            = "example-bot"
  location            = "global"
  resource_group_name = "rg"

  site {
    name                            = "default"
    enabled                         = true
    enhanced_authentication_enabled = true
  }
}

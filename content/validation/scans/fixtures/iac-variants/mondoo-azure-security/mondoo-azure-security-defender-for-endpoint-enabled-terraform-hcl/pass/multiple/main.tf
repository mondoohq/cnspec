resource "azurerm_security_center_setting" "wdatp" {
  setting_name = "WDATP"
  enabled      = true
}

resource "azurerm_security_center_setting" "mcas" {
  setting_name = "MCAS"
  enabled      = false
}

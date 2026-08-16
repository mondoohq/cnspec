resource "azurerm_security_center_setting" "wdatp" {
  setting_name = "WDATP"
  enabled      = false
}

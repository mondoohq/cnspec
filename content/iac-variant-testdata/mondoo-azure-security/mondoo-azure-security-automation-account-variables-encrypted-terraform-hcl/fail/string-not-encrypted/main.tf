resource "azurerm_automation_variable_string" "fail" {
  name                    = "example-var"
  resource_group_name     = "example-rg"
  automation_account_name = "example-automation"
  value                   = "secret-value"
  encrypted               = false
}

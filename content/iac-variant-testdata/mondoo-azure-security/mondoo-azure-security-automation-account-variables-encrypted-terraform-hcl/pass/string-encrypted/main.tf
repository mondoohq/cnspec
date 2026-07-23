resource "azurerm_automation_variable_string" "pass" {
  name                    = "example-var"
  resource_group_name     = "example-rg"
  automation_account_name = "example-automation"
  value                   = "secret-value"
  encrypted               = true
}

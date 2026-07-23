resource "azurerm_automation_variable_int" "fail_int" {
  name                    = "example-int-var"
  resource_group_name     = "example-rg"
  automation_account_name = "example-automation"
  value                   = 42
}

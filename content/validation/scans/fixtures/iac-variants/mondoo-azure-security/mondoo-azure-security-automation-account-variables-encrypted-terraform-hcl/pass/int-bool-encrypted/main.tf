resource "azurerm_automation_variable_int" "pass_int" {
  name                    = "example-int-var"
  resource_group_name     = "example-rg"
  automation_account_name = "example-automation"
  value                   = 42
  encrypted               = true
}

resource "azurerm_automation_variable_bool" "pass_bool" {
  name                    = "example-bool-var"
  resource_group_name     = "example-rg"
  automation_account_name = "example-automation"
  value                   = true
  encrypted               = true
}

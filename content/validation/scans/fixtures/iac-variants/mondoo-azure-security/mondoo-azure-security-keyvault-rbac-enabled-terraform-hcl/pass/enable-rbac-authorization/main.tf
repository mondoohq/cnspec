resource "azurerm_key_vault" "pass" {
  name                = "example-kv"
  location            = "eastus"
  resource_group_name = "example-rg"
  tenant_id           = "00000000-0000-0000-0000-000000000000"
  sku_name            = "standard"

  enable_rbac_authorization = true
}

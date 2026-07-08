resource "azurerm_key_vault_managed_hardware_security_module" "example" {
  name                          = "example-hsm"
  resource_group_name           = "example-rg"
  location                      = "eastus"
  sku_name                      = "Standard_B1"
  tenant_id                     = "00000000-0000-0000-0000-000000000000"
  admin_object_ids              = ["11111111-1111-1111-1111-111111111111"]
  purge_protection_enabled      = true
  soft_delete_retention_days    = 90
  public_network_access_enabled = true
}

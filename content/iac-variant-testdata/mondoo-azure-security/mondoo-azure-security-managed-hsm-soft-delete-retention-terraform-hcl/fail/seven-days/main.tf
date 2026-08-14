# A seven-day window gives very little time to notice and reverse a malicious
# or mistaken deletion of the HSM.
resource "azurerm_key_vault_managed_hardware_security_module" "prod" {
  name                       = "prod-hsm"
  resource_group_name        = azurerm_resource_group.prod.name
  location                   = azurerm_resource_group.prod.location
  sku_name                   = "Standard_B1"
  tenant_id                  = var.tenant_id
  admin_object_ids           = [var.hsm_admin_object_id]
  soft_delete_retention_days = 7
}

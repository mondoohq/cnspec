# An omitted retention period takes the 90-day default.
resource "azurerm_key_vault_managed_hardware_security_module" "prod" {
  name                = "prod-hsm"
  resource_group_name = azurerm_resource_group.prod.name
  location            = azurerm_resource_group.prod.location
  sku_name            = "Standard_B1"
  tenant_id           = var.tenant_id
  admin_object_ids    = [var.hsm_admin_object_id]
}

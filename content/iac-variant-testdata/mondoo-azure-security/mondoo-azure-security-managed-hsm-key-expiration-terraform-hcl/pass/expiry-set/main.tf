resource "azurerm_key_vault_managed_hardware_security_module_key" "signing" {
  name           = "payment-signing"
  managed_hsm_id = azurerm_key_vault_managed_hardware_security_module.prod.id
  key_type       = "EC-HSM"
  curve          = "P-256"
  key_opts       = ["sign", "verify"]
  expiration_date = "2027-01-01T00:00:00Z"
}

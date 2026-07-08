resource "azurerm_storage_container" "example" {
  name                              = "example-container"
  storage_account_name              = azurerm_storage_account.example.name
  container_access_type             = "private"
  default_encryption_scope          = azurerm_storage_encryption_scope.example.name
  encryption_scope_override_enabled = true
}

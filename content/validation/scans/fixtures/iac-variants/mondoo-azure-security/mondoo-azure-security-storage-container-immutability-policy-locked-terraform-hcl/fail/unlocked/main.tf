resource "azurerm_storage_container_immutability_policy" "example" {
  storage_container_resource_manager_id = azurerm_storage_container.example.resource_manager_id
  immutability_period_in_days           = 14
  locked                                = false
}

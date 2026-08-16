resource "azurerm_managed_disk" "example" {
  name                   = "disk-example"
  location               = "eastus"
  resource_group_name    = "example-rg"
  storage_account_type   = "Premium_LRS"
  create_option          = "Empty"
  disk_size_gb           = 128
  disk_encryption_set_id = azurerm_disk_encryption_set.example.id
}

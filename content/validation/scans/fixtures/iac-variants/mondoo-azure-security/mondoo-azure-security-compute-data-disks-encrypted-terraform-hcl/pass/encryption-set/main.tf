resource "azurerm_managed_disk" "example" {
  name                   = "example-data-disk"
  location               = "eastus"
  resource_group_name    = "example-rg"
  storage_account_type   = "Premium_LRS"
  create_option          = "Empty"
  disk_size_gb           = 128
  disk_encryption_set_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Compute/diskEncryptionSets/example-des"
}

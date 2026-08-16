# Compliant: the OS disk references a disk encryption set, which is what sets the
# disk's encryption type to EncryptionAtRestWithCustomerKey.
resource "azurerm_linux_virtual_machine" "pass_example" {
  name                = "pass-example"
  resource_group_name = "example-rg"
  location            = "eastus"
  size                = "Standard_F2"
  admin_username      = "adminuser"

  network_interface_ids = ["/subscriptions/x/nic"]

  os_disk {
    caching                = "ReadWrite"
    storage_account_type   = "Premium_LRS"
    disk_encryption_set_id = "/subscriptions/x/resourceGroups/example-rg/providers/Microsoft.Compute/diskEncryptionSets/des1"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }
}

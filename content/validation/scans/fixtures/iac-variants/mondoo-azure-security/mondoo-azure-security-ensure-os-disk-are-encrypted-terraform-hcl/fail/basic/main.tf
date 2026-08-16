# Non-compliant: the OS disk names no disk encryption set, so it stays on the
# platform-managed key. encryption_at_host_enabled covers the temp disk and host
# caches only, and does not change the OS disk's encryption type.
resource "azurerm_linux_virtual_machine" "fail_example" {
  name                       = "fail-example"
  resource_group_name        = "example-rg"
  location                   = "eastus"
  size                       = "Standard_F2"
  admin_username             = "adminuser"
  encryption_at_host_enabled = true

  network_interface_ids = ["/subscriptions/x/nic"]

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }
}

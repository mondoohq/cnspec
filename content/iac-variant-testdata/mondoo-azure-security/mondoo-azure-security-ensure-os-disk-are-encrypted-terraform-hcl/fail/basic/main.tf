# Non-compliant: Linux VM does not enable encryption at host.
resource "azurerm_linux_virtual_machine" "fail_example" {
  name                       = "fail-example"
  resource_group_name        = "example-rg"
  location                   = "eastus"
  size                       = "Standard_F2"
  admin_username             = "adminuser"
  encryption_at_host_enabled = false

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

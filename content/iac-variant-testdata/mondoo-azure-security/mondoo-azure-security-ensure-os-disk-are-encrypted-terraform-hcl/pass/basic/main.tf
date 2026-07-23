# Compliant: Linux VM enables encryption at host.
resource "azurerm_linux_virtual_machine" "pass_example" {
  name                       = "pass-example"
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

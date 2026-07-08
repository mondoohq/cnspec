resource "azurerm_linux_virtual_machine" "example" {
  name                = "example-linux-vm"
  resource_group_name = "example-rg"
  location            = "eastus"
  size                = "Standard_DS1_v2"
  admin_username      = "adminuser"

  network_interface_ids = [
    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/networkInterfaces/example-nic",
  ]

  admin_ssh_key {
    username   = "adminuser"
    public_key = "ssh-rsa AAAAB3Nza..."
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }
}

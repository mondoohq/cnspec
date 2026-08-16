resource "azurerm_windows_virtual_machine" "example" {
  name                  = "winvm"
  resource_group_name   = "example-rg"
  location              = "eastus"
  size                  = "Standard_B1s"
  admin_username        = "adminuser"
  admin_password        = "P@ssw0rd1234!"
  network_interface_ids = [azurerm_network_interface.example.id]

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2022-Datacenter"
    version   = "latest"
  }
}

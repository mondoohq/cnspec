# Disable Password Authentication
resource "azurerm_linux_virtual_machine" "disable_password_authentication_bad_1" {
  name                            = "bad-linux-machine"
  resource_group_name             = azurerm_resource_group.example.name
  location                        = azurerm_resource_group.example.location
  size                            = "Standard_F2"
  admin_username                  = "adminuser"
  admin_password                  = "somePassword"
  disable_password_authentication = false
}

resource "azurerm_virtual_machine" "disable_password_authentication_bad_2" {
	name                            = "bad-linux-machine"
	resource_group_name             = azurerm_resource_group.example.name
	location                        = azurerm_resource_group.example.location
	size                            = "Standard_F2"
	admin_username                  = "adminuser"
	admin_password                  = "somePassword"

	os_profile {
		computer_name  = "hostname"
		admin_username = "testadmin"
		admin_password = "Password1234!"
	}

	os_profile_linux_config {
		disable_password_authentication = false
	}
}

# Disk Encryption

resource "azurerm_managed_disk" "bad_example" {
  encryption_settings {
    enabled = false
  }
}

# Managed Disk public network access

resource "azurerm_managed_disk" "bad_example" {
  network_access_policy = "AllowAll"
}

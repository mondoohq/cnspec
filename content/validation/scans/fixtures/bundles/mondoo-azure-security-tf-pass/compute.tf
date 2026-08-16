# Disable Password Authentication
resource "azurerm_linux_virtual_machine" "disable_password_authentication_good_1" {
  name                            = "good-linux-machine"
  resource_group_name             = azurerm_resource_group.example.name
  location                        = azurerm_resource_group.example.location
  size                            = "Standard_F2"
  admin_username                  = "adminuser"
  admin_password                  = "somePassword"
  
  admin_ssh_key {
    username   = "adminuser"
    public_key = file("~/.ssh/id_rsa.pub")
  }
}

resource "azurerm_virtual_machine" "disable_password_authentication_good_2" {
	name                            = "good-linux-machine"
	resource_group_name             = azurerm_resource_group.example.name
	location                        = azurerm_resource_group.example.location
	size                            = "Standard_F2"
	admin_username                  = "adminuser"

	
	os_profile_linux_config {
		ssh_keys = [{
			key_data = file("~/.ssh/id_rsa.pub")
			path = "~/.ssh/id_rsa.pub"
		}]

		disable_password_authentication = true
	}
}

# Disk Encryption

resource "azurerm_managed_disk" "good_example" {
  encryption_settings {
    enabled = true
  }
}

# Managed Disk public network access

resource "azurerm_managed_disk" "good_example" {
  network_access_policy = "DenyAll"
}
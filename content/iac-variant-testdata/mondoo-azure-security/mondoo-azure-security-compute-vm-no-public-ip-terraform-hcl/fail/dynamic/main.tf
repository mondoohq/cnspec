variable "ip_configs" {
  type = list(object({
    name      = string
    subnet_id = string
    public_ip = string
  }))
  default = [
    {
      name      = "external"
      subnet_id = "subnet-1"
      public_ip = "pip-1"
    }
  ]
}

resource "azurerm_network_interface" "example" {
  name                = "nic-example"
  location            = "eastus"
  resource_group_name = "example-rg"

  dynamic "ip_configuration" {
    for_each = var.ip_configs
    content {
      name                          = ip_configuration.value.name
      subnet_id                     = ip_configuration.value.subnet_id
      private_ip_address_allocation = "Dynamic"
      public_ip_address_id          = ip_configuration.value.public_ip
    }
  }
}

resource "azurerm_batch_pool" "example" {
  name                = "examplepool"
  resource_group_name = "example-rg"
  account_name        = "examplebatchaccount"
  display_name        = "Example Pool"
  vm_size             = "Standard_A1_v2"
  node_agent_sku_id   = "batch.node.ubuntu 22.04"

  fixed_scale {
    target_dedicated_nodes = 1
  }

  storage_image_reference {
    publisher = "canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }
}

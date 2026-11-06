resource "azurerm_machine_learning_compute_cluster" "example" {
  name                          = "example-cluster"
  location                      = "eastus"
  vm_priority                   = "LowPriority"
  vm_size                       = "Standard_DS2_v2"
  machine_learning_workspace_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.MachineLearningServices/workspaces/ws"
  ssh_public_access_enabled     = true
  node_public_ip_enabled        = false
  subnet_resource_id            = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/sn"

  scale_settings {
    min_node_count                       = 0
    max_node_count                       = 5
    scale_down_nodes_after_idle_duration = "PT30S"
  }
}

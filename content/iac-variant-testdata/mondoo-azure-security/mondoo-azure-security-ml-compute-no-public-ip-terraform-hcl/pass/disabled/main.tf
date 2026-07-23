resource "azurerm_machine_learning_compute_cluster" "pass" {
  name                          = "example-cluster"
  location                      = "eastus"
  vm_priority                   = "LowPriority"
  vm_size                       = "Standard_DS2_v2"
  machine_learning_workspace_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.MachineLearningServices/workspaces/ws"
  subnet_resource_id            = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/sn"
  node_public_ip_enabled        = false

  scale_settings {
    min_node_count                       = 0
    max_node_count                       = 5
    scale_down_nodes_after_idle_duration = "PT30S"
  }
}

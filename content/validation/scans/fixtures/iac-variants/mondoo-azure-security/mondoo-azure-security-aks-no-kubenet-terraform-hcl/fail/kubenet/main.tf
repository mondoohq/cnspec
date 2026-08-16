resource "azurerm_kubernetes_cluster" "example" {
  name                = "example-aks"
  location            = "eastus"
  resource_group_name = "example-rg"
  dns_prefix          = "exampleaks"

  network_profile {
    network_plugin = "kubenet"
    pod_cidr       = "10.244.0.0/16"
  }

  default_node_pool {
    name       = "default"
    node_count = 2
    vm_size    = "Standard_D2_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}

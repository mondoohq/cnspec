resource "azurerm_kubernetes_cluster" "example" {
  name                = "example-aks"
  location            = "eastus"
  resource_group_name = "example-rg"
  dns_prefix          = "exampleaks"

  default_node_pool {
    name                    = "default"
    node_count              = 3
    vm_size                 = "Standard_D2s_v3"
    host_encryption_enabled = true
  }

  identity {
    type = "SystemAssigned"
  }
}

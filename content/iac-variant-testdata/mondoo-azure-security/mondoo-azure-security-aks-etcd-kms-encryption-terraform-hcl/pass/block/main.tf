resource "azurerm_kubernetes_cluster" "example" {
  name                = "example-aks"
  location            = "eastus"
  resource_group_name = "example-rg"
  dns_prefix          = "exampleaks"

  default_node_pool {
    name       = "default"
    node_count = 3
    vm_size    = "Standard_D2s_v3"
  }

  azure_key_vault_kms {
    enabled  = true
    key_vault_key_id = azurerm_key_vault_key.example.id
  }

  identity {
    type = "SystemAssigned"
  }
}
